package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var godotExePattern = regexp.MustCompile(`(?i)^godot.*\.exe$`)
var godotFeatureVersion = regexp.MustCompile(`config/features=PackedStringArray\("([^"]+)"`)
var godotReleaseVersion = regexp.MustCompile(`^([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)

// scanGodotEditors finds portable Godot executables in the configured roots
// and the usual install locations. Godot has no registry-wide installer, so
// the executable name and --version are the source of truth.
func scanGodotEditors(roots []string) []editorInstall {
	candidates := dedupe(append(append([]string{}, roots...), knownGodotDirs()...))
	seen := map[string]bool{}
	var out []editorInstall
	for _, root := range candidates {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			continue
		}
		baseDepth := strings.Count(filepath.Clean(root), string(os.PathSeparator))
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				depth := strings.Count(filepath.Clean(path), string(os.PathSeparator)) - baseDepth
				if depth > 3 || path != root && (skipProjectDir(d.Name()) || strings.EqualFold(d.Name(), "Editor")) {
					return filepath.SkipDir
				}
				return nil
			}
			name := d.Name()
			if !godotExePattern.MatchString(name) || strings.Contains(strings.ToLower(name), "console") {
				return nil
			}
			abs, err := filepath.Abs(path)
			if err != nil || seen[strings.ToLower(abs)] {
				return nil
			}
			version := godotEditorVersion(abs, name)
			if version == "" {
				return nil
			}
			seen[strings.ToLower(abs)] = true
			out = append(out, editorInstall{
				Version: version, DisplayVersion: version, Kind: kindGodot,
				Exe: abs, Root: filepath.Dir(abs),
			})
			return nil
		})
	}
	return out
}

func knownGodotDirs() []string {
	var dirs []string
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA", "USERPROFILE"} {
		if root := os.Getenv(env); root != "" {
			dirs = append(dirs,
				filepath.Join(root, "Godot"),
				filepath.Join(root, "Programs", "Godot"),
				filepath.Join(root, "scoop", "apps", "godot"),
				filepath.Join(root, "Microsoft", "WinGet", "Packages"),
			)
		}
	}
	return dirs
}

func godotEditorVersion(exe, filename string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "--version")
	hideWindowCmd(cmd)
	if out, err := cmd.Output(); err == nil {
		if fields := strings.Fields(string(out)); len(fields) > 0 {
			if v := godotReleaseVersion.FindString(fields[0]); v != "" {
				return v
			}
			return fields[0]
		}
	}
	if m := regexp.MustCompile(`(?i)godot[_-]?v?([0-9]+\.[0-9]+(?:\.[0-9]+)?)`).FindStringSubmatch(filename); m != nil {
		return m[1]
	}
	return ""
}

// scanGodotProjects finds project.godot directories under the configured
// project roots. Godot project versions are the engine feature version, not
// a Unity ProjectVersion.txt.
func scanGodotProjects(p prefs, editors []editorInstall) []projectEntry {
	roots := dedupe(append(defaultProjectRoots(), p.ProjectRoots...))
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
	var out []projectEntry
	for _, root := range roots {
		if root == "" {
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
			if !isGodotProjectDir(path) {
				return nil
			}
			key := projectIdentityKey(path)
			if hidden[key] {
				return filepath.SkipDir
			}
			version, modified := readGodotProject(path)
			entry := projectEntry{
				Path: path, Name: filepath.Base(path), Version: version,
				DisplayVersion: version, Kind: kindGodot, Modified: modified,
				Favorite: favorites[key],
			}
			for _, e := range editors {
				if e.Kind == kindGodot && (version == "" || strings.HasPrefix(version, e.Version) || strings.HasPrefix(e.Version, version)) {
					entry.Editor = e.Exe
					break
				}
			}
			out = append(out, entry)
			return filepath.SkipDir
		})
	}
	return out
}

func readGodotProject(dir string) (string, time.Time) {
	path := filepath.Join(dir, "project.godot")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", time.Time{}
	}
	version := ""
	if m := godotFeatureVersion.FindSubmatch(data); m != nil {
		if v := godotReleaseVersion.FindString(string(m[1])); v != "" {
			version = v
		}
	}
	modified := time.Time{}
	if info, err := os.Stat(path); err == nil {
		modified = info.ModTime()
	}
	return version, modified
}

func isGodotProjectDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "project.godot"))
	return err == nil && !info.IsDir()
}

func supportsInstallFlow(kind string) bool {
	return kind == kindUnity || kind == kindTuanjie
}

func supportsProjectCreation(kind string) bool {
	if a := engineAdapterByID(kind); a != nil {
		return a.Features.CreateProject
	}
	return false
}

func supportsEditorManagement(kind string) bool {
	return kind == kindUnity || kind == kindTuanjie
}
