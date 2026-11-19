package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"unitystarter/locales"
)

func TestGlobalSettingsEntryIsPinnedAtRailBottom(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	global, ok := tt.Find(locales.SidebarGlobalSettings())
	if !ok {
		t.Fatalf("global settings entry is missing: %q", tt.Texts())
	}
	if global.X < 0 || global.X+global.W > engineRailWidth || global.Y < 600 || global.Y+global.H > 760 {
		t.Fatalf("global settings is not pinned at the rail bottom: %+v", global)
	}
	if err := tt.Click(locales.SidebarGlobalSettings()); err != nil {
		t.Fatal(err)
	}
	if a.page != pageGlobalSettings {
		t.Fatalf("global settings entry opened %q", a.page)
	}
}

func TestGlobalSettingsListsEveryEngine(t *testing.T) {
	a := fixture()
	a.page = pageGlobalSettings
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText(locales.GlobalSettingsAllEngines()) {
		t.Fatalf("global settings heading: %q", tt.Texts())
	}
	for _, adapter := range engineAdapters() {
		detail := strings.TrimSpace(adapter.Description)
		if detail == "" {
			detail = adapter.ID
		}
		if !tt.HasText(detail) {
			t.Fatalf("global settings omits %s: %q", adapter.ID, tt.Texts())
		}
	}
	if !tt.HasText(locales.EngineDownloadWorkers()) || !tt.HasText(locales.EngineInstallRoot()) {
		t.Fatalf("global settings omits engine controls: %q", tt.Texts())
	}
	if _, ok := tt.Find(locales.EngineSettings()); !ok {
		t.Fatalf("global settings has no engine settings action: %q", tt.Texts())
	}
}

func TestSwitchEngineKeepsGlobalSettingsPage(t *testing.T) {
	a := fixture()
	a.page = pageGlobalSettings
	a.switchEngine(kindTuanjie)
	if a.page != pageGlobalSettings {
		t.Fatalf("switching engines left global settings for %q", a.page)
	}
}
