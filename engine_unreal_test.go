package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"unitystarter/locales"
)

func writeUnrealTestEditor(t *testing.T, root, version string) string {
	t.Helper()
	var rel string
	switch runtime.GOOS {
	case "windows":
		rel = filepath.Join("Engine", "Binaries", "Win64", "UnrealEditor.exe")
	case "darwin":
		rel = filepath.Join("Engine", "Binaries", "Mac", "UnrealEditor.app", "Contents", "MacOS", "UnrealEditor")
	default:
		rel = filepath.Join("Engine", "Binaries", "Linux", "UnrealEditor")
	}
	exe := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(version, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	buildDir := filepath.Join(root, "Engine", "Build")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"MajorVersion":%s,"MinorVersion":%s,"PatchVersion":%s}`, parts[0], parts[1], parts[2])
	if err := os.WriteFile(filepath.Join(buildDir, "Build.version"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return exe
}

func TestUnrealEditorAndProjectScan(t *testing.T) {
	root := t.TempDir()
	engineRoot := filepath.Join(root, "UE_5.4")
	exe := writeUnrealTestEditor(t, engineRoot, "5.4.4")
	manifest := filepath.Join(root, "LauncherInstalled.dat")
	data, err := json.Marshal(map[string]any{
		"InstallationList": []map[string]any{{
			"InstallLocation": engineRoot,
			"AppName":         "UE_5.4",
			"AppVersion":      "5.4.4-123+UE5",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}

	editors := scanUnrealEditorsIn(nil, manifest)
	if len(editors) != 1 {
		t.Fatalf("editors: %+v", editors)
	}
	if editors[0].Kind != kindUnreal || editors[0].Version != "5.4.4" || !strings.EqualFold(editors[0].Exe, exe) {
		t.Fatalf("unreal editor: %+v", editors[0])
	}

	projectsRoot := filepath.Join(root, "Projects")
	projectDir := filepath.Join(projectsRoot, "MyGame")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	uproject := filepath.Join(projectDir, "MyGame.uproject")
	if err := os.WriteFile(uproject, []byte(`{"EngineAssociation":"5.4"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	projects := scanUnrealProjects(prefs{ProjectRoots: []string{projectsRoot}}, editors, false)
	var found *projectEntry
	for i := range projects {
		if strings.EqualFold(projects[i].Path, projectDir) {
			found = &projects[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("unreal project missing: %+v", projects)
	}
	if found.Kind != kindUnreal || found.Name != "MyGame" || found.Version != "5.4" || found.Editor != exe {
		t.Fatalf("unreal project: %+v", *found)
	}
	if found.Modified.IsZero() {
		t.Fatalf("unreal project has no modification time: %+v", *found)
	}
}

func TestUnrealVersionMatchingAndNewestEditor(t *testing.T) {
	if !unrealVersionMatches("5.4", "5.4.4") || !unrealVersionMatches("5.4.4", "5.4") {
		t.Fatal("5.4 should match 5.4.4 in both directions")
	}
	if unrealVersionMatches("5.5", "5.4.4") {
		t.Fatal("5.5 must not match 5.4.4")
	}
	if got := normalizeUnrealVersion("5.4.4-123+UE5"); got != "5.4.4" {
		t.Fatalf("normalized version: %q", got)
	}
	editors := []editorInstall{
		{Kind: kindUnreal, Version: "5.4.2", Exe: "old"},
		{Kind: kindUnreal, Version: "5.4.4", Exe: "new"},
	}
	if got := unrealEditorExeForVersion("5.4", editors); got != "new" {
		t.Fatalf("newest matching editor: %q", got)
	}
}

// The startup path reloads the registry from disk. It must keep every
// built-in engine, including the ones added after the original three.
func TestStartupReloadKeepsEveryBuiltinEngine(t *testing.T) {
	withCleanAdapterRegistry(t)
	if errs := reloadEngineAdapters(t.TempDir()); len(errs) != 0 {
		t.Fatalf("reload errors: %v", errs)
	}
	for _, id := range []string{kindUnity, kindTuanjie, kindGodot, kindUnreal} {
		if engineAdapterByID(id) == nil {
			t.Fatalf("startup reload lost built-in engine %q", id)
		}
	}
}

func TestScanUnrealEditorsReadsEpicManifestFromProgramData(t *testing.T) {
	programData := filepath.Join(t.TempDir(), "ProgramData")
	engineRoot := filepath.Join(programData, "UE_5.4")
	exe := writeUnrealTestEditor(t, engineRoot, "5.4.4")
	manifest := filepath.Join(programData, "Epic", "UnrealEngineLauncher", "LauncherInstalled.dat")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]any{
		"InstallationList": []map[string]any{{
			"InstallLocation": engineRoot,
			"AppName":         "UE_5.4",
			"AppVersion":      "5.4.4-123+UE5",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ProgramData", programData)

	editors := scanUnrealEditors(nil)
	for _, editor := range editors {
		if strings.EqualFold(editor.Exe, exe) {
			if editor.Kind != kindUnreal || editor.Version != "5.4.4" {
				t.Fatalf("manifest editor: %+v", editor)
			}
			return
		}
	}
	t.Fatalf("manifest editor is missing: %+v", editors)
}

func TestNonInstallableInstallsPageExplainsItsScanRoots(t *testing.T) {
	for _, tc := range []struct {
		name   string
		engine string
		page   string
		hint   string
	}{
		{"unreal", kindUnreal, pageUnrealInstalls, locales.InstallsEmptyHintUnreal()},
		{"godot", kindGodot, pageGodotInstalls, locales.InstallsEmptyHintGodot()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := fixture()
			a.activeEngine, a.page = tc.engine, tc.page
			tt := ui.NewTester(a.view, 1180, 760)
			if !tt.HasText(tc.hint) {
				t.Fatalf("%s installs page hint: %q", tc.name, tt.Texts())
			}
			if _, ok := tt.Find(locales.ActionAddFolder()); !ok {
				t.Fatalf("%s installs page is missing Add folder: %q", tc.name, tt.Texts())
			}
		})
	}
}

func TestPinnedUnrealAndGodotProjectsAreFound(t *testing.T) {
	root := t.TempDir()
	unrealDir := filepath.Join(root, "UEGame")
	if err := os.MkdirAll(unrealDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unrealDir, "UEGame.uproject"), []byte(`{"EngineAssociation":"5.4"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	godotDir := filepath.Join(root, "GodotGame")
	if err := os.MkdirAll(godotDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(godotDir, "project.godot"), []byte("config/features=PackedStringArray(\"4.3\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	unreal := scanUnrealProjects(prefs{PinnedProjects: []string{unrealDir}}, nil, false)
	if len(unreal) != 1 || !strings.EqualFold(unreal[0].Path, unrealDir) {
		t.Fatalf("pinned Unreal project: %+v", unreal)
	}
	godot := scanGodotProjects(prefs{PinnedProjects: []string{godotDir}}, nil)
	if len(godot) != 1 || !strings.EqualFold(godot[0].Path, godotDir) {
		t.Fatalf("pinned Godot project: %+v", godot)
	}
}

func TestUnrealProjectsPageOffersManualAdd(t *testing.T) {
	a := fixture()
	a.activeEngine, a.page = kindUnreal, pageUnrealProjects
	tt := ui.NewTester(a.view, 1180, 760)
	if _, ok := tt.Find(locales.ActionAddProject()); !ok {
		t.Fatalf("Unreal projects page is missing Add project: %q", tt.Texts())
	}
}

func TestUnrealMissingEditorOpensOfficialPage(t *testing.T) {
	a := fixture()
	a.openProject(projectEntry{Path: t.TempDir(), Name: "Future", Version: "5.5", Kind: kindUnreal})
	if !a.missingEditorOpen || a.missingDownloadFound {
		t.Fatalf("missing editor prompt: open=%v found=%v", a.missingEditorOpen, a.missingDownloadFound)
	}
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText(locales.MissingEditorTitle()) {
		t.Fatalf("missing editor dialog: %q", tt.Texts())
	}
	a.openDownloadsForMissingEditor()
	if a.page != pageUnrealGet {
		t.Fatalf("missing editor opened %q", a.page)
	}
}

func TestEngineHubDownloadPages(t *testing.T) {
	for _, tc := range []struct {
		name   string
		engine string
		page   string
		texts  []string
	}{
		{"unity", kindUnity, pageUnityGet, []string{locales.UnityHubDownload(), locales.UnityArchive(), locales.UnityHubDocs()}},
		{"tuanjie", kindTuanjie, pageTuanjieGet, []string{locales.TuanjieHubDownload(), locales.TuanjieManual(), locales.UnityHubDownload()}},
		{"godot", kindGodot, pageGodotGet, []string{locales.GodotDownload(), locales.GodotAssetLibrary()}},
		{"unreal", kindUnreal, pageUnrealGet, []string{locales.UnrealLauncherDownload(), locales.UnrealEngineDownload()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := fixture()
			a.activeEngine, a.page = tc.engine, tc.page
			tt := ui.NewTester(a.view, 1180, 760)
			for _, text := range tc.texts {
				if !tt.HasText(text) {
					t.Fatalf("%s page is missing %q: %q", tc.name, text, tt.Texts())
				}
			}
		})
	}
}

func TestGetPagesShowHubStatus(t *testing.T) {
	a := fixture()
	a.hubs = []hub{{Kind: kindUnity, Dir: `C:\UnityHub`, Version: "3.12.0", Exe: `C:\UnityHub\Unity Hub.exe`}}
	a.activeEngine, a.page = kindUnity, pageUnityGet
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText("3.12.0") || !tt.HasText(locales.ActionOpenHub()) {
		t.Fatalf("installed Hub status: %q", tt.Texts())
	}

	missing := fixture()
	missing.hubs = nil
	missing.activeEngine, missing.page = kindTuanjie, pageTuanjieGet
	tt = ui.NewTester(missing.view, 1180, 760)
	if !tt.HasText(locales.HubNotDetected()) || !tt.HasText(locales.HubNotDetectedHint()) {
		t.Fatalf("missing Hub status: %q", tt.Texts())
	}
}

func TestHubDownloadEntriesAreInNavigation(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	if _, ok := tt.Find(locales.UnityGet()); !ok {
		t.Fatalf("Unity Hub entry is missing: %q", tt.Texts())
	}
	if err := tt.Click(locales.UnityGet()); err != nil {
		t.Fatal(err)
	}
	if a.page != pageUnityGet {
		t.Fatalf("Unity Hub entry opened %q", a.page)
	}
	a.switchEngine(kindTuanjie)
	tt.Frame()
	if _, ok := tt.Find(locales.TuanjieGet()); !ok {
		t.Fatalf("Tuanjie Hub entry is missing: %q", tt.Texts())
	}
	if err := tt.Click(locales.TuanjieGet()); err != nil {
		t.Fatal(err)
	}
	if a.page != pageTuanjieGet {
		t.Fatalf("Tuanjie Hub entry opened %q", a.page)
	}
}
