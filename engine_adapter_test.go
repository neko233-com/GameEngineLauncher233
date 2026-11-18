package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"unitystarter/locales"
)

const testAdapterManifest = `{
  "schemaVersion": 1,
  "id": "myengine",
  "name": "My Engine",
  "version": "0.2.0",
  "description": "Adapter used by the native tests",
  "homepage": "https://example.com/myengine",
  "iconSvg": "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 24 24\"><rect x=\"3\" y=\"3\" width=\"18\" height=\"18\" rx=\"4\" fill=\"currentColor\"/></svg>",
  "pages": [
    {"id": "hub", "title": "My Engine Hub", "kind": "links", "icon": "projects", "groups": [
      {"title": "My Engine", "links": [{"title": "Engine docs", "detail": "Reference and guides", "url": "https://example.com/docs"}]}
    ]},
    {"id": "community", "title": "Community", "kind": "links", "icon": "feedback", "groups": [
      {"title": "Community", "links": [{"title": "Forum", "url": "https://example.com/forum"}]}
    ]}
  ]
}`

func withCleanAdapterRegistry(t *testing.T) {
	t.Helper()
	resetEngineRegistry()
	t.Cleanup(resetEngineRegistry)
}

func TestEngineNavigationDiffersByAdapter(t *testing.T) {
	withCleanAdapterRegistry(t)
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	if _, ok := tt.Find(locales.PagePublish()); ok {
		t.Fatal("Unity adapter must not offer game publishing")
	}
	if err := tt.Click(channelName(kindTuanjie)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find(locales.PagePublish()); !ok {
		t.Fatalf("Tuanjie adapter lost game publishing: %q", tt.Texts())
	}
	if err := tt.Click(channelName(kindUnity)); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find(locales.PagePublish()); ok {
		t.Fatal("Unity adapter still offers game publishing")
	}
}

func TestExternalAdapterDrivesRailAndPages(t *testing.T) {
	withCleanAdapterRegistry(t)
	dir := t.TempDir()
	if _, err := installEngineAdapterManifest(dir, []byte(testAdapterManifest), "https://example.com/myengine/adapter.json"); err != nil {
		t.Fatal(err)
	}
	if errs := reloadEngineAdapters(dir); len(errs) != 0 {
		t.Fatalf("reload: %v", errs)
	}
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	if _, ok := tt.Find("My Engine"); !ok {
		t.Fatalf("imported engine is missing from the rail: %q", tt.Texts())
	}
	if err := tt.Click("My Engine"); err != nil {
		t.Fatal(err)
	}
	if a.activeEngine != "myengine" || a.page != externalPageRoute("myengine", "hub") {
		t.Fatalf("switching to the adapter left engine=%q page=%q", a.activeEngine, a.page)
	}
	if !tt.HasText("My Engine Hub") || !tt.HasText("Engine docs") || !tt.HasText("Community") {
		t.Fatalf("adapter links page content: %q", tt.Texts())
	}
	if _, ok := tt.Find(locales.SidebarInstalls()); ok {
		t.Fatal("links-only adapter inherited Unity's installs navigation")
	}
	if _, ok := tt.Find(locales.SidebarDownloads()); ok {
		t.Fatal("links-only adapter inherited the pinned downloads entry")
	}
	if path := os.Getenv("ADAPTER_SNAPSHOT"); path != "" {
		if err := writePNG(path, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestImportAdapterFromURLAndRemove(t *testing.T) {
	withCleanAdapterRegistry(t)
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/adapter.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(testAdapterManifest))
	}))
	defer server.Close()
	adapter, err := importEngineAdapter(context.Background(), server.Client(), server.URL, dir)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.ID != "myengine" || len(adapter.Pages) != 2 {
		t.Fatalf("imported adapter: %+v", adapter)
	}
	if _, err := os.Stat(filepath.Join(dir, "myengine", "adapter.json")); err != nil {
		t.Fatalf("adapter was not written: %v", err)
	}
	if errs := reloadEngineAdapters(dir); len(errs) != 0 || engineAdapterByID("myengine") == nil {
		t.Fatalf("imported adapter was not registered: %v", errs)
	}
	if err := removeInstalledEngineAdapter(dir, "myengine"); err != nil {
		t.Fatal(err)
	}
	reloadEngineAdapters(dir)
	if engineAdapterByID("myengine") != nil {
		t.Fatal("removed adapter is still registered")
	}
}

func TestAdapterManifestRejectsUnsafeValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"reserved id", strings.Replace(testAdapterManifest, `"myengine"`, `"unity"`, 1)},
		{"traversal id", strings.Replace(testAdapterManifest, `"myengine"`, `"../escape"`, 1)},
		{"bad url", strings.Replace(testAdapterManifest, `https://example.com/docs`, `file:///etc/passwd`, 1)},
		{"unknown kind", strings.Replace(testAdapterManifest, `"kind": "links"`, `"kind": "shell"`, 1)},
		{"no pages", `{"schemaVersion":1,"id":"empty","name":"Empty","pages":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseEngineAdapterManifest([]byte(tc.body), "", ""); err == nil {
				t.Fatal("unsafe manifest was accepted")
			}
		})
	}
}

func TestSettingsShowsEngineAdapters(t *testing.T) {
	withCleanAdapterRegistry(t)
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	if err := tt.Click(locales.SidebarSettings()); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(locales.SettingsEnginesTab()); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(locales.EnginesTitle()) || !tt.HasText(locales.EnginesImport()) || !tt.HasText(locales.ChannelUnity()) {
		t.Fatalf("engine adapter settings: %q", tt.Texts())
	}
}

func TestExternalAdapterProcessDetectAndLaunch(t *testing.T) {
	withCleanAdapterRegistry(t)
	marker := filepath.Join(t.TempDir(), "launched.txt")
	t.Setenv("GO_WANT_ENGINE_ADAPTER_HELPER", "1")
	t.Setenv("ENGINE_ADAPTER_HELPER_MARKER", marker)
	adapter := &engineAdapter{
		ID: "helper", Version: "0.1.0", BuiltIn: false,
		name: func() string { return "Helper Engine" },
		Pages: []enginePage{{
			Key: "hub", Route: externalPageRoute("helper", "hub"), Kind: "projects", Icon: iconProjects,
			title: func() string { return "Helper" },
		}},
		Process: &engineAdapterProcess{
			Command: os.Args[0],
			Args:    []string{"-test.run=TestEngineAdapterHelperProcess", "--"},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := detectEngineAdapter(ctx, adapter)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Editors) != 1 || result.Editors[0].Version != "1.2.3" || len(result.Projects) != 1 {
		t.Fatalf("detect result: %+v", result)
	}
	editors, projects := scanExternalEngineAdapters(ctx, []*engineAdapter{adapter})
	if len(editors) != 1 || len(projects) != 1 || projects[0].Kind != "helper" || projects[0].Editor == "" {
		t.Fatalf("scan merge: editors=%+v projects=%+v", editors, projects)
	}
	if err := launchViaEngineAdapter(ctx, adapter, editors[0].Exe, `C:\Fake\Project`, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != `C:\Fake\Project` {
		t.Fatalf("launch marker=%q err=%v", string(data), err)
	}
	a := fixture()
	engineRegistryMu.Lock()
	addEngineAdapterLocked(adapter)
	engineRegistryMu.Unlock()
	a.editors = []editorInstall{{Kind: "helper", Exe: `C:\Fake\engine.exe`, Version: "1.2.3"}}
	project := projectEntry{Kind: "helper", Path: `C:\Fake\Project`, Name: "Fake", Version: "1.2.3", Editor: `C:\Fake\engine.exe`}
	a.openExternalProject(project, a.editors[0])
	data, err = os.ReadFile(marker)
	if err != nil || string(data) != `C:\Fake\Project` {
		t.Fatalf("project launch marker=%q err=%v", string(data), err)
	}
}

// TestEngineAdapterHelperProcess is the child side of the LSP-like process
// test: it speaks Content-Length JSON-RPC on stdin/stdout.
func TestEngineAdapterHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_ENGINE_ADAPTER_HELPER") != "1" {
		return
	}
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout
	for {
		length := -1
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if strings.HasPrefix(strings.ToLower(line), "content-length:") {
				length, _ = strconv.Atoi(strings.TrimSpace(strings.SplitN(line, ":", 2)[1]))
			}
		}
		if length < 0 {
			return
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(reader, data); err != nil {
			return
		}
		var req struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(data, &req) != nil {
			return
		}
		var result any = map[string]any{}
		switch req.Method {
		case "detect":
			result = map[string]any{
				"editors": []any{map[string]any{
					"version": "1.2.3", "displayVersion": "1.2.3",
					"exe": `C:\Fake\engine.exe`, "root": `C:\Fake`,
				}},
				"projects": []any{map[string]any{
					"path": `C:\Fake\Project`, "name": "Fake Project", "version": "1.2.3",
				}},
			}
		case "launch":
			var params struct {
				ProjectPath string `json:"projectPath"`
			}
			_ = json.Unmarshal(req.Params, &params)
			_ = os.WriteFile(os.Getenv("ENGINE_ADAPTER_HELPER_MARKER"), []byte(params.ProjectPath), 0o600)
		case "shutdown":
			writeAdapterHelperResponse(writer, req.ID, result)
			return
		}
		writeAdapterHelperResponse(writer, req.ID, result)
	}
}

func writeAdapterHelperResponse(w io.Writer, id int, result any) {
	data, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(data))
	_, _ = w.Write(data)
}

func TestNormalizeAdapterURL(t *testing.T) {
	for raw, want := range map[string]string{
		"owner/repo":                                         "https://raw.githubusercontent.com/owner/repo/HEAD/adapter.json",
		"https://github.com/owner/repo":                      "https://raw.githubusercontent.com/owner/repo/HEAD/adapter.json",
		"https://github.com/owner/repo/tree/dev":             "https://raw.githubusercontent.com/owner/repo/dev/adapter.json",
		"https://github.com/owner/repo/tree/dev/sub":         "https://raw.githubusercontent.com/owner/repo/dev/sub/adapter.json",
		"https://github.com/owner/repo/blob/dev/custom.json": "https://raw.githubusercontent.com/owner/repo/dev/custom.json",
	} {
		got, err := normalizeAdapterURL(raw)
		if err != nil || got != want {
			t.Errorf("normalize %q = %q, %v; want %q", raw, got, err, want)
		}
	}
	if _, err := normalizeAdapterURL("ftp://example.com/adapter.json"); err == nil {
		t.Fatal("ftp adapter URL was accepted")
	}
}
