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

func TestGlobalSettingsTileStaysPinnedWithManyEngines(t *testing.T) {
	addRailTestEngines(t)
	a := fixture()
	tt := ui.NewTester(a.view, 900, 560)
	before, ok := tt.Find(locales.SidebarGlobalSettings())
	if !ok || before.Y < 500 || before.Y+before.H > 560 {
		t.Fatalf("global settings tile is not pinned: %+v found=%v", before, ok)
	}
	tt.Scroll(engineRailWidth/2, 200, 0, 2000)
	tt.Frame()
	after, ok := tt.Find(locales.SidebarGlobalSettings())
	if !ok || after.X != before.X || after.Y != before.Y || after.W != before.W || after.H != before.H {
		t.Fatalf("scrolling engines moved global settings: before=%+v after=%+v", before, after)
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
