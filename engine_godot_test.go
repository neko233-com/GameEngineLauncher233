package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"unitystarter/locales"
)

func TestGodotAdapterNavigationAndActions(t *testing.T) {
	withCleanAdapterRegistry(t)
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	if err := tt.Click("Godot"); err != nil {
		t.Fatal(err)
	}
	if a.activeEngine != kindGodot || a.page != pageGodotProjects {
		t.Fatalf("Godot tab left engine=%q page=%q", a.activeEngine, a.page)
	}
	if _, ok := tt.Find(locales.PagePublish()); ok {
		t.Fatal("Godot must not inherit game publishing")
	}
	if _, ok := tt.Find(locales.GodotGet()); !ok {
		t.Fatalf("Godot links are missing: %q", tt.Texts())
	}
	if err := tt.Click(locales.SidebarInstalls()); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find(locales.InstallEditor()); ok {
		t.Fatal("Godot has no built-in editor installer")
	}
	if _, ok := tt.Find(locales.SidebarDownloads()); ok {
		t.Fatal("Godot must not show the pinned downloads entry")
	}
	if _, ok := tt.Find(locales.SidebarLicenseLabel() + ":"); ok {
		t.Fatal("Godot must not show a license status")
	}
}

func TestEngineAndSystemSettingsAreSeparate(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	if err := tt.Click(locales.EngineSettings()); err != nil {
		t.Fatal(err)
	}
	if a.page != pageEngineSettings {
		t.Fatalf("engine gear opened %q", a.page)
	}
	if !tt.HasText(locales.EngineInstallRoot()) || !tt.HasText(locales.EngineDownloadWorkers()) {
		t.Fatalf("engine settings content: %q", tt.Texts())
	}
	if err := tt.Click(locales.SidebarSettings()); err != nil {
		t.Fatal(err)
	}
	if a.page != pageSettings {
		t.Fatalf("system settings opened %q", a.page)
	}
	if !tt.HasText(locales.SettingsLanguage()) || !tt.HasText(locales.SettingsEnginesTab()) {
		t.Fatalf("system settings content: %q", tt.Texts())
	}
}

func TestGodotEditorAndProjectScan(t *testing.T) {
	root := t.TempDir()
	editorDir := filepath.Join(root, "Godot")
	if err := os.MkdirAll(editorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(editorDir, "Godot_v4.3-stable_win64.exe")
	if err := os.WriteFile(exe, []byte("not a real executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	projectFile := "config_version=5\n\n[application]\n\nconfig/name=\"demo\"\nconfig/features=PackedStringArray(\"4.3\", \"Forward Plus\")\n"
	if err := os.WriteFile(filepath.Join(projectDir, "project.godot"), []byte(projectFile), 0o644); err != nil {
		t.Fatal(err)
	}
	editors := scanGodotEditors([]string{root})
	if len(editors) != 1 || editors[0].Kind != kindGodot || editors[0].Version != "4.3" {
		t.Fatalf("Godot editors: %+v", editors)
	}
	projects := scanGodotProjects(prefs{ProjectRoots: []string{root}}, editors)
	if len(projects) != 1 || projects[0].Kind != kindGodot || projects[0].Version != "4.3" || projects[0].Editor == "" {
		t.Fatalf("Godot projects: %+v", projects)
	}
}

func TestEngineSpecificInstallRootsAndWorkers(t *testing.T) {
	a := newApp()
	unityRoot := filepath.Join(t.TempDir(), "unity")
	tuanjieRoot := filepath.Join(t.TempDir(), "tuanjie")
	a.setInstallRootFor(kindUnity, unityRoot)
	a.setInstallRootFor(kindTuanjie, tuanjieRoot)
	if a.installRootFor(kindUnity) != unityRoot || a.installRootFor(kindTuanjie) != tuanjieRoot {
		t.Fatalf("install roots are not independent: unity=%q tuanjie=%q",
			a.installRootFor(kindUnity), a.installRootFor(kindTuanjie))
	}
	if a.downloadWorkers(kindUnity) != parallelWorkers {
		t.Fatalf("default workers = %d", a.downloadWorkers(kindUnity))
	}
	a.setDownloadWorkers(kindUnity, 2)
	a.setDownloadWorkers(kindTuanjie, 4)
	if a.downloadWorkers(kindUnity) != 2 || a.downloadWorkers(kindTuanjie) != 4 {
		t.Fatalf("workers are not independent: unity=%d tuanjie=%d",
			a.downloadWorkers(kindUnity), a.downloadWorkers(kindTuanjie))
	}
}
