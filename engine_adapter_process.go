package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// engineAdapterProtocolVersion is the JSON-RPC handshake version. The framing
// is LSP's: Content-Length headers, then one JSON object per message.
const engineAdapterProtocolVersion = 1

type adapterRPC struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr bytes.Buffer
	mu     sync.Mutex
	nextID int
}

type adapterRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type adapterRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      int              `json:"id"`
	Result  json.RawMessage  `json:"result"`
	Error   *adapterRPCError `json:"error"`
}

type adapterDetectResult struct {
	Editors  []adapterDetectedEditor  `json:"editors"`
	Projects []adapterDetectedProject `json:"projects"`
}

type adapterDetectedEditor struct {
	Version        string         `json:"version"`
	DisplayVersion string         `json:"displayVersion,omitempty"`
	Exe            string         `json:"exe"`
	Root           string         `json:"root,omitempty"`
	Modules        []editorModule `json:"modules,omitempty"`
}

type adapterDetectedProject struct {
	Path     string `json:"path"`
	Name     string `json:"name,omitempty"`
	Version  string `json:"version,omitempty"`
	Modified string `json:"modified,omitempty"`
}

func startEngineAdapterProcess(ctx context.Context, adapter *engineAdapter) (*adapterRPC, error) {
	if adapter == nil || adapter.Process == nil {
		return nil, fmt.Errorf("adapter has no process")
	}
	if adapter.Process.Command == "" {
		return nil, fmt.Errorf("adapter process command is empty")
	}
	cmd := exec.CommandContext(ctx, adapter.Process.Command, adapter.Process.Args...)
	cmd.Dir = adapter.Dir
	hideWindowCmd(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	r := &adapterRPC{cmd: cmd, stdin: stdin, stdout: bufio.NewReaderSize(stdout, 64<<10), nextID: 1}
	cmd.Stderr = &r.stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start adapter process: %w", err)
	}
	return r, nil
}

func (r *adapterRPC) call(ctx context.Context, method string, params, result any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	id := r.nextID
	r.nextID++
	request := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		request["params"] = params
	}
	if err := r.writeMessage(request); err != nil {
		return err
	}
	done := make(chan struct{})
	var response adapterRPCResponse
	var readErr error
	go func() {
		defer close(done)
		for {
			response, readErr = r.readResponse()
			if readErr != nil {
				return
			}
			if response.ID != id {
				continue
			}
			return
		}
	}()
	select {
	case <-ctx.Done():
		_ = r.cmd.Process.Kill()
		return ctx.Err()
	case <-done:
		if readErr != nil {
			return readErr
		}
		if response.Error != nil {
			return fmt.Errorf("adapter %s: %s", method, response.Error.Message)
		}
		if result != nil && len(response.Result) > 0 {
			if err := json.Unmarshal(response.Result, result); err != nil {
				return fmt.Errorf("adapter %s result: %w", method, err)
			}
		}
		return nil
	}
}

func (r *adapterRPC) writeMessage(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(r.stdin, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	_, err = r.stdin.Write(data)
	return err
}

func (r *adapterRPC) readResponse() (adapterRPCResponse, error) {
	length := -1
	for {
		line, err := r.stdout.ReadString('\n')
		if err != nil {
			return adapterRPCResponse{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			continue
		}
		length, err = strconv.Atoi(strings.TrimSpace(value))
		if err != nil || length < 0 || length > 4<<20 {
			return adapterRPCResponse{}, fmt.Errorf("adapter sent an invalid Content-Length")
		}
	}
	if length < 0 {
		return adapterRPCResponse{}, fmt.Errorf("adapter sent no Content-Length")
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r.stdout, data); err != nil {
		return adapterRPCResponse{}, err
	}
	var response adapterRPCResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return adapterRPCResponse{}, fmt.Errorf("adapter sent invalid JSON: %w", err)
	}
	return response, nil
}

func (r *adapterRPC) initialize(ctx context.Context, adapter *engineAdapter) error {
	params := map[string]any{
		"protocolVersion": engineAdapterProtocolVersion,
		"client":          map[string]any{"name": "UnityStarter", "version": appVersion},
		"engine":          map[string]any{"id": adapter.ID, "name": adapter.DisplayName()},
	}
	var result map[string]any
	return r.call(ctx, "initialize", params, &result)
}

func (r *adapterRPC) close() {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	_ = r.call(ctx, "shutdown", nil, nil)
	_ = r.stdin.Close()
	done := make(chan struct{})
	go func() {
		_ = r.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = r.cmd.Process.Kill()
		<-done
	}
}

// detectEngineAdapter starts a process adapter, initializes it and asks for
// editors and projects. A broken adapter is isolated to its own error.
func detectEngineAdapter(ctx context.Context, adapter *engineAdapter) (adapterDetectResult, error) {
	var result adapterDetectResult
	if adapter == nil || adapter.Process == nil {
		return result, fmt.Errorf("adapter has no process")
	}
	callCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	rpc, err := startEngineAdapterProcess(callCtx, adapter)
	if err != nil {
		return result, err
	}
	defer rpc.close()
	if err := rpc.initialize(callCtx, adapter); err != nil {
		return result, err
	}
	if err := rpc.call(callCtx, "detect", map[string]any{}, &result); err != nil {
		return result, err
	}
	return result, nil
}

// launchViaEngineAdapter asks a process adapter to launch a project or the
// editor itself. The adapter owns its CLI conventions.
func launchViaEngineAdapter(ctx context.Context, adapter *engineAdapter, exe, projectPath string, args []string) error {
	if adapter == nil || adapter.Process == nil {
		return fmt.Errorf("adapter has no launch process")
	}
	callCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	rpc, err := startEngineAdapterProcess(callCtx, adapter)
	if err != nil {
		return err
	}
	defer rpc.close()
	if err := rpc.initialize(callCtx, adapter); err != nil {
		return err
	}
	params := map[string]any{"exe": exe, "projectPath": projectPath, "args": args}
	return rpc.call(callCtx, "launch", params, nil)
}

// scanExternalEngineAdapters merges every process adapter's detected editors
// and projects into the normal scan result.
func scanExternalEngineAdapters(ctx context.Context, adapters []*engineAdapter) ([]editorInstall, []projectEntry) {
	var editors []editorInstall
	var projects []projectEntry
	for _, adapter := range adapters {
		if adapter.BuiltIn || adapter.Process == nil {
			continue
		}
		result, err := detectEngineAdapter(ctx, adapter)
		if err != nil {
			logger.Printf("adapter %s: detect: %v", adapter.ID, err)
			continue
		}
		for _, e := range result.Editors {
			if e.Exe == "" || e.Version == "" {
				continue
			}
			editors = append(editors, editorInstall{
				Version:        e.Version,
				DisplayVersion: e.DisplayVersion,
				Kind:           adapter.ID,
				Exe:            e.Exe,
				Root:           e.Root,
				Modules:        e.Modules,
			})
		}
		for _, p := range result.Projects {
			if p.Path == "" {
				continue
			}
			name := p.Name
			if name == "" {
				name = filepathBase(p.Path)
			}
			modified := time.Time{}
			if p.Modified != "" {
				if t, err := time.Parse(time.RFC3339, p.Modified); err == nil {
					modified = t
				}
			}
			entry := projectEntry{
				Path: p.Path, Name: name, Version: p.Version,
				DisplayVersion: p.Version, Kind: adapter.ID, Modified: modified,
			}
			for _, e := range editors {
				if e.Kind == adapter.ID && e.Version == p.Version {
					entry.Editor = e.Exe
					break
				}
			}
			projects = append(projects, entry)
		}
	}
	return editors, projects
}

func filepathBase(path string) string {
	path = strings.TrimRight(strings.ReplaceAll(path, "\\", "/"), "/")
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
