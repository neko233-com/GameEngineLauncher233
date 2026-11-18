package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Unreal Engine is an editor ecosystem without a Hub the launcher writes to.
// We discover installed engines from the Epic Games Launcher manifest, the
// usual Epic Games folders and any editor root the user configured, then read
// each engine's Engine/Build/Build.version. Projects are found by their
// .uproject file and associated by the EngineAssociation field.
var unrealVersionPattern = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)

// unrealInstallHint is one engine the Epic Games Launcher records.
type unrealInstallHint struct {
	Location string
	Version  string
}

// unrealProjectInfo is what a .uproject says before an editor is chosen.
type unrealProjectInfo struct {
	File        string
	Name        string
	Association string
	Modified    time.Time
}

func epicLauncherManifestPath() string {
	if dir := strings.TrimSpace(os.Getenv("ProgramData")); dir != "" {
		return filepath.Join(dir, "Epic", "UnrealEngineLauncher", "LauncherInstalled.dat")
	}
	return ""
}

func scanUnrealEditors(roots []string) []editorInstall {
	candidates := append([]string{}, roots...)
	candidates = append(candidates, knownUnrealDirs()...)
	candidates = append(candidates, unrealRegistryDirs()...)
	return scanUnrealEditorsIn(candidates, epicLauncherManifestPath())
}

// scanUnrealEditorsIn is scanUnrealEditors with explicit roots and a
// LauncherInstalled.dat path, so tests can exercise the manifest without the
// machine's own Epic data.
func scanUnrealEditorsIn(roots []string, manifest string) []editorInstall {
	hints := readEpicLauncherManifest(manifest)
	candidates := make([]string, 0, len(roots)+len(hints))
	for _, hint := range hints {
		candidates = append(candidates, hint.Location)
	}
	for _, root := range dedupe(roots) {
		candidates = append(candidates, unrealCandidateDirs(root)...)
	}

	seen := map[string]bool{}
	var out []editorInstall
	for _, dir := range candidates {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		hint := ""
		for _, h := range hints {
			if sameInstallPath(h.Location, dir) {
				hint = h.Version
				break
			}
		}
		editor, ok := unrealEditorAt(dir, hint)
		if !ok {
			continue
		}
		key := strings.ToLower(filepath.Clean(editor.Exe))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, editor)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if c := compareVersions(out[i].Version, out[j].Version); c != 0 {
			return c > 0
		}
		return strings.ToLower(out[i].Exe) < strings.ToLower(out[j].Exe)
	})
	return out
}

func readEpicLauncherManifest(path string) []unrealInstallHint {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var manifest struct {
		InstallationList []struct {
			InstallLocation string `json:"InstallLocation"`
			AppName         string `json:"AppName"`
			AppVersion      string `json:"AppVersion"`
		} `json:"InstallationList"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return nil
	}
	var out []unrealInstallHint
	for _, item := range manifest.InstallationList {
		location := strings.TrimSpace(item.InstallLocation)
		if location == "" {
			continue
		}
		version := normalizeUnrealVersion(item.AppVersion)
		if version == "" {
			version = normalizeUnrealVersion(item.AppName)
		}
		out = append(out, unrealInstallHint{Location: location, Version: version})
	}
	return out
}

func knownUnrealDirs() []string {
	var dirs []string
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
		if root := os.Getenv(env); root != "" {
			dirs = append(dirs, filepath.Join(root, "Epic Games"))
		}
	}
	return dirs
}

// unrealCandidateDirs treats root as an engine folder, a folder containing
// many engines, or an "Epic Games" folder with UE_* children.
func unrealCandidateDirs(root string) []string {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}
	out := []string{root}
	entries, err := os.ReadDir(root)
	if err != nil {
		return out
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		child := filepath.Join(root, entry.Name())
		out = append(out, child)
		if strings.EqualFold(entry.Name(), "Epic Games") {
			subs, err := os.ReadDir(child)
			if err != nil {
				continue
			}
			for _, sub := range subs {
				if sub.IsDir() {
					out = append(out, filepath.Join(child, sub.Name()))
				}
			}
		}
	}
	return out
}

func unrealEditorAt(dir, hint string) (editorInstall, bool) {
	exe := unrealEditorExe(dir)
	if exe == "" {
		return editorInstall{}, false
	}
	version := readUnrealBuildVersion(dir)
	if version == "" {
		version = normalizeUnrealVersion(hint)
	}
	if version == "" {
		version = normalizeUnrealVersion(filepath.Base(dir))
	}
	if version == "" {
		version = "unknown"
	}
	return editorInstall{
		Version:        version,
		DisplayVersion: version,
		Kind:           kindUnreal,
		Exe:            exe,
		Root:           dir,
	}, true
}

func unrealEditorExe(root string) string {
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		candidates = []string{
			filepath.Join(root, "Engine", "Binaries", "Win64", "UnrealEditor.exe"),
			filepath.Join(root, "Engine", "Binaries", "Win64", "UE4Editor.exe"),
		}
	case "darwin":
		candidates = []string{
			filepath.Join(root, "Engine", "Binaries", "Mac", "UnrealEditor.app", "Contents", "MacOS", "UnrealEditor"),
			filepath.Join(root, "Engine", "Binaries", "Mac", "UE4Editor.app", "Contents", "MacOS", "UE4Editor"),
		}
	default:
		candidates = []string{
			filepath.Join(root, "Engine", "Binaries", "Linux", "UnrealEditor"),
			filepath.Join(root, "Engine", "Binaries", "Linux", "UE4Editor"),
		}
	}
	for _, exe := range candidates {
		info, err := os.Stat(exe)
		if err != nil || info.IsDir() {
			continue
		}
		if abs, err := filepath.Abs(exe); err == nil {
			return abs
		}
		return exe
	}
	return ""
}

func readUnrealBuildVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "Engine", "Build", "Build.version"))
	if err != nil {
		return ""
	}
	var build struct {
		MajorVersion int `json:"MajorVersion"`
		MinorVersion int `json:"MinorVersion"`
		PatchVersion int `json:"PatchVersion"`
	}
	if json.Unmarshal(data, &build) != nil || build.MajorVersion <= 0 {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d", build.MajorVersion, build.MinorVersion, build.PatchVersion)
}

// normalizeUnrealVersion reads 5.4, 5.4.4 or an AppVersion such as
// 5.4.4-123+UE5 and returns the dotted version.
func normalizeUnrealVersion(value string) string {
	m := unrealVersionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return ""
	}
	version := m[1] + "." + m[2]
	if m[3] != "" {
		version += "." + m[3]
	}
	return version
}

// scanUnrealProjects finds .uproject projects under the same configured roots
// as Unity and Godot. EngineAssociation is the project's engine line; a project
// built from source may carry a GUID, which will not match a launcher install.
func scanUnrealProjects(p prefs, editors []editorInstall, withVCS bool) []projectEntry {
	roots := dedupe(append(defaultProjectRoots(), p.ProjectRoots...))
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		roots = append(roots, filepath.Join(home, "Documents", "Unreal Projects"))
	}
	for _, dir := range p.PinnedProjects {
		if isUnrealProjectDir(dir) {
			roots = append(roots, dir)
		}
	}
	roots = dedupe(roots)

	hidden := map[string]bool{}
	for key, path := range p.HiddenProjects {
		hidden[projectIdentityKey(key)] = true
		hidden[projectIdentityKey(path)] = true
	}
	favorites := map[string]bool{}
	for path, favorite := range p.LocalFavorites {
		if favorite {
			favorites[projectIdentityKey(path)] = true
		}
	}

	seen := map[string]bool{}
	var out []projectEntry
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		baseDepth := strings.Count(filepath.Clean(root), string(os.PathSeparator))
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			depth := strings.Count(filepath.Clean(path), string(os.PathSeparator)) - baseDepth
			if depth > 4 || path != root && skipProjectDir(d.Name()) {
				return filepath.SkipDir
			}
			info, ok := readUnrealProject(path)
			if !ok {
				return nil
			}
			key := projectIdentityKey(path)
			if hidden[key] || seen[key] {
				return filepath.SkipDir
			}
			seen[key] = true
			entry := projectEntry{
				Path:     path,
				Name:     info.Name,
				Version:  info.Association,
				Kind:     kindUnreal,
				Modified: info.Modified,
				Favorite: favorites[key],
			}
			entry.Editor = unrealEditorExeForVersion(entry.Version, editors)
			if withVCS {
				entry.VCS = detectVSCCached(path)
			}
			out = append(out, entry)
			return filepath.SkipDir
		})
	}
	return out
}

func isUnrealProjectDir(dir string) bool {
	_, ok := unrealProjectFile(dir)
	return ok
}

func unrealProjectFile(dir string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".uproject") {
			continue
		}
		return filepath.Join(dir, entry.Name()), true
	}
	return "", false
}

func readUnrealProject(dir string) (unrealProjectInfo, bool) {
	path, ok := unrealProjectFile(dir)
	if !ok {
		return unrealProjectInfo{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return unrealProjectInfo{}, false
	}
	var project struct {
		EngineAssociation string `json:"EngineAssociation"`
	}
	_ = json.Unmarshal(data, &project)
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if name == "" {
		name = filepath.Base(dir)
	}
	modified := time.Time{}
	if info, err := os.Stat(path); err == nil {
		modified = info.ModTime()
	}
	return unrealProjectInfo{
		File:        path,
		Name:        name,
		Association: strings.TrimSpace(project.EngineAssociation),
		Modified:    modified,
	}, true
}

// unrealEditorExeForVersion chooses the newest installed engine matching a
// project's EngineAssociation. "5.4" matches "5.4.4"; an empty association
// accepts any installed Unreal editor.
func unrealEditorExeForVersion(version string, editors []editorInstall) string {
	var best editorInstall
	for _, editor := range editors {
		if editor.Kind != kindUnreal || editor.Exe == "" {
			continue
		}
		if version != "" && !unrealVersionMatches(version, editor.Version) {
			continue
		}
		if best.Exe == "" || compareVersions(editor.Version, best.Version) > 0 {
			best = editor
		}
	}
	return best.Exe
}

func unrealVersionMatches(project, editor string) bool {
	project = normalizeUnrealVersion(project)
	editor = normalizeUnrealVersion(editor)
	if project == "" || editor == "" {
		return project == "" || editor == ""
	}
	if strings.EqualFold(project, editor) {
		return true
	}
	return strings.HasPrefix(editor, project+".") || strings.HasPrefix(project, editor+".")
}

func sameInstallPath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
