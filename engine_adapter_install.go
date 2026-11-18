package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// adaptersDir is where imported engine adapters live, one folder per id.
func adaptersDir() string {
	return filepath.Join(userDataDir(), "engine-adapters")
}

// engineAdapterSpecURL is the published specification and example.
func engineAdapterSpecURL() string {
	return urlRepo + "/blob/main/ENGINE_ADAPTERS.md"
}

// normalizeAdapterURL accepts a GitHub repository, a GitHub file/tree URL,
// an owner/repo shorthand or a direct manifest URL.
func normalizeAdapterURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("enter a GitHub repository or adapter.json URL")
	}
	if strings.HasPrefix(raw, "github.com/") {
		raw = "https://" + raw
	}
	if !strings.Contains(raw, "://") {
		parts := strings.Split(strings.TrimSuffix(raw, "/"), "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			raw = "https://github.com/" + parts[0] + "/" + parts[1]
		} else {
			return "", fmt.Errorf("use owner/repo or an https URL")
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", fmt.Errorf("use an http(s) URL")
	}
	if strings.EqualFold(u.Host, "github.com") {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			return "", fmt.Errorf("GitHub URL must name a repository")
		}
		owner, repo := parts[0], parts[1]
		repo = strings.TrimSuffix(repo, ".git")
		switch {
		case len(parts) >= 5 && parts[2] == "blob":
			branch := parts[3]
			path := strings.Join(parts[4:], "/")
			return "https://raw.githubusercontent.com/" + owner + "/" + repo + "/" + branch + "/" + path, nil
		case len(parts) >= 4 && parts[2] == "tree":
			branch := parts[3]
			path := strings.Join(parts[4:], "/")
			if path == "" {
				path = "adapter.json"
			} else if !strings.HasSuffix(strings.ToLower(path), ".json") {
				path = strings.TrimSuffix(path, "/") + "/adapter.json"
			}
			return "https://raw.githubusercontent.com/" + owner + "/" + repo + "/" + branch + "/" + path, nil
		default:
			return "https://raw.githubusercontent.com/" + owner + "/" + repo + "/HEAD/adapter.json", nil
		}
	}
	if strings.HasSuffix(u.Path, "/") || u.Path == "" {
		u.Path = strings.TrimSuffix(u.Path, "/") + "/adapter.json"
	}
	return u.String(), nil
}

// importEngineAdapter downloads and validates an adapter manifest, then
// installs it under dir. It is deliberately manifest-only: a process adapter
// may name a binary inside its installed folder, but importing never executes
// anything.
func importEngineAdapter(ctx context.Context, client *http.Client, rawURL, dir string) (*engineAdapter, error) {
	manifestURL, err := normalizeAdapterURL(rawURL)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("adapter request: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/plain;q=0.9")
	req.Header.Set("User-Agent", "GameEngineLauncher233/"+appVersion)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download adapter: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download adapter: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAdapterManifestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read adapter: %w", err)
	}
	if len(body) > maxAdapterManifestBytes {
		return nil, fmt.Errorf("adapter manifest is larger than %d KiB", maxAdapterManifestBytes/1024)
	}
	return installEngineAdapterManifest(dir, body, manifestURL)
}

func installEngineAdapterManifest(dir string, body []byte, source string) (*engineAdapter, error) {
	probe, err := parseEngineAdapterManifest(body, source, "")
	if err != nil {
		return nil, err
	}
	target := filepath.Join(dir, probe.ID)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return nil, fmt.Errorf("create adapter folder: %w", err)
	}
	adapter, err := parseEngineAdapterManifest(body, source, target)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(filepath.Join(target, "adapter.json"), adapterManifestBytes(body)); err != nil {
		return nil, fmt.Errorf("write adapter manifest: %w", err)
	}
	if err := writeFileAtomic(filepath.Join(target, "source.txt"), []byte(source+"\n")); err != nil {
		return nil, fmt.Errorf("write adapter source: %w", err)
	}
	return adapter, nil
}

// loadInstalledEngineAdapters reads every <dir>/<id>/adapter.json. A broken
// manifest is reported and skipped; it never blocks the other adapters.
func loadInstalledEngineAdapters(dir string) ([]*engineAdapter, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []error{err}
	}
	var out []*engineAdapter
	var errs []error
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		root := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filepath.Join(root, "adapter.json"))
		if err != nil {
			if !os.IsNotExist(err) {
				errs = append(errs, fmt.Errorf("%s: %w", entry.Name(), err))
			}
			continue
		}
		source := ""
		if raw, err := os.ReadFile(filepath.Join(root, "source.txt")); err == nil {
			source = strings.TrimSpace(string(raw))
		}
		adapter, err := parseEngineAdapterManifest(data, source, root)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", entry.Name(), err))
			continue
		}
		if adapter.ID != entry.Name() {
			errs = append(errs, fmt.Errorf("%s: adapter id %q does not match its folder", entry.Name(), adapter.ID))
			continue
		}
		out = append(out, adapter)
	}
	return out, errs
}

// reloadEngineAdapters replaces the user half of the registry and returns
// any manifests that failed to load. Built-ins always remain.
func reloadEngineAdapters(dir string) []error {
	user, errs := loadInstalledEngineAdapters(dir)
	engineRegistryMu.Lock()
	engineRegistryByID = map[string]*engineAdapter{}
	engineRegistryOrder = nil
	for _, a := range builtinEngineAdapters() {
		engineRegistryByID[a.ID] = a
		engineRegistryOrder = append(engineRegistryOrder, a.ID)
	}
	for _, a := range user {
		addEngineAdapterLocked(a)
	}
	engineRegistryLoaded = time.Now()
	engineRegistryMu.Unlock()
	return errs
}

// removeInstalledEngineAdapter deletes one imported adapter and refreshes the
// registry. Built-ins are never removable.
func removeInstalledEngineAdapter(dir, id string) error {
	if !engineAdapterIDPattern.MatchString(id) || engineAdapterReserved[id] {
		return fmt.Errorf("that engine is built in and cannot be removed")
	}
	target := filepath.Join(dir, id)
	root, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(target)
	if err != nil || !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return fmt.Errorf("refusing to remove a path outside the adapter folder")
	}
	if err := os.RemoveAll(abs); err != nil {
		return err
	}
	return nil
}
