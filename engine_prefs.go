package main

import (
	"path/filepath"
	"strings"
)

// enginePrefs are settings that belong to one engine. System settings stay
// launcher-wide; install roots and download concurrency follow the engine.
type enginePrefs struct {
	InstallRoot     string `json:"installRoot,omitempty"`
	DownloadWorkers int    `json:"downloadWorkers,omitempty"`
}

func (a *app) enginePrefsFor(kind string) enginePrefs {
	if a.prefs.EngineSettings == nil {
		return enginePrefs{}
	}
	return a.prefs.EngineSettings[kind]
}

func (a *app) setEnginePrefs(kind string, p enginePrefs) {
	if a.prefs.EngineSettings == nil {
		a.prefs.EngineSettings = map[string]enginePrefs{}
	}
	a.prefs.EngineSettings[kind] = p
}

// installRoot is the active engine's root, kept for callers that already work
// in the active-engine context.
func (a *app) installRoot() string {
	return a.installRootFor(a.activeEngine)
}

func (a *app) installRootFor(kind string) string {
	if p := a.enginePrefsFor(kind); strings.TrimSpace(p.InstallRoot) != "" {
		return p.InstallRoot
	}
	// Settings from before per-engine roots were introduced apply to every
	// engine, so upgrades keep discovering existing installs.
	if strings.TrimSpace(a.prefs.InstallRoot) != "" {
		return a.prefs.InstallRoot
	}
	return defaultInstallRootFor(kind)
}

func (a *app) setInstallRootFor(kind, dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	p := a.enginePrefsFor(kind)
	previous := a.installRootFor(kind)
	if previous != "" && !strings.EqualFold(previous, dir) {
		a.prefs.EditorRoots = dedupe(append(a.prefs.EditorRoots, previous))
	}
	p.InstallRoot = dir
	a.setEnginePrefs(kind, p)
}

func (a *app) downloadWorkers(kind string) int {
	n := a.enginePrefsFor(kind).DownloadWorkers
	if n < 1 {
		return parallelWorkers
	}
	return min(n, parallelWorkers)
}

func (a *app) setDownloadWorkers(kind string, n int) {
	n = min(max(n, 1), parallelWorkers)
	p := a.enginePrefsFor(kind)
	p.DownloadWorkers = n
	a.setEnginePrefs(kind, p)
	a.save()
}

// defaultInstallRootFor separates engines that used to share one root. The
// legacy UnityStarter root remains in the scan roots for upgrades.
func defaultInstallRootFor(kind string) string {
	base := defaultInstallRoot()
	switch kind {
	case kindUnity:
		return filepath.Join(base, "Unity")
	case kindTuanjie:
		return filepath.Join(base, "Tuanjie")
	case "godot":
		return filepath.Join(base, "Godot")
	default:
		if kind == "" {
			return base
		}
		return filepath.Join(base, "Engines", kind)
	}
}
