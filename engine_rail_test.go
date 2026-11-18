package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"unitystarter/locales"
)

// The engine switcher is a vertical rail at the far left, as Tuanjie Hub's:
// both engines stack, the active one is raised, and the navigation column
// begins to its right.
func TestEngineRailIsVerticalAndSwitches(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)

	unity, ok := tt.Find(channelName(kindUnity))
	if !ok {
		t.Fatal("the Unity tab is missing from the rail")
	}
	tuanjie, ok := tt.Find(channelName(kindTuanjie))
	if !ok {
		t.Fatal("the Tuanjie tab is missing from the rail")
	}
	const railWidth = 64
	if unity.X != 0 || unity.W != railWidth || unity.H < 40 {
		t.Fatalf("Unity tab is not a rail tab: %+v", unity)
	}
	if tuanjie.X != 0 || tuanjie.W != railWidth {
		t.Fatalf("Tuanjie tab is not a rail tab: %+v", tuanjie)
	}
	if tuanjie.Y < unity.Y+unity.H {
		t.Fatalf("engine tabs are not stacked vertically: unity=%+v tuanjie=%+v", unity, tuanjie)
	}

	projects, ok := tt.Find(locales.SidebarProjects())
	if !ok || projects.X < railWidth {
		t.Fatalf("navigation overlaps the engine rail: %+v, found=%v", projects, ok)
	}
	gear, ok := tt.Find(locales.SidebarSettings())
	if !ok || gear.Y+gear.H > 120 {
		t.Fatalf("settings gear is not in the navigation header: %+v, found=%v", gear, ok)
	}
	license, ok := tt.Find(locales.SidebarLicenseLabel() + ":")
	if !ok {
		t.Fatal("license caption is missing")
	}
	downloads, ok := tt.Find(locales.SidebarDownloads())
	if !ok {
		t.Fatal("downloads entry is missing")
	}
	if downloads.Y < license.Y {
		t.Fatalf("downloads is not pinned below the license caption: downloads=%+v license=%+v", downloads, license)
	}
	if downloads.Y+downloads.H > 760 {
		t.Fatalf("downloads escaped the window: %+v", downloads)
	}

	// The active tab has a raised tile; the other sits on the rail's own
	// background. Sampling inside the tile's left edge avoids the icon.
	img := tt.Image()
	active := img.RGBAAt(int(unity.X)+14, int(unity.Y)+28)
	idle := img.RGBAAt(int(tuanjie.X)+14, int(tuanjie.Y)+28)
	if active.R <= idle.R+0x10 {
		t.Fatalf("active engine tab has no raised tile: active=%+v idle=%+v", active, idle)
	}

	if err := tt.Click(channelName(kindTuanjie)); err != nil {
		t.Fatal(err)
	}
	if a.activeEngine != kindTuanjie || a.page != pageTuanjieProjects {
		t.Fatalf("rail click left engine=%q page=%q", a.activeEngine, a.page)
	}
	if !tt.HasText(locales.ChannelTuanjie()) {
		t.Fatalf("navigation header did not follow the rail: %q", tt.Texts())
	}
}

// Downloads keeps its Tuanjie Hub position at the foot of the column while
// the navigable destinations above it scroll independently.
func TestEngineRailPinsDownloadsWhileNavigationScrolls(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 900, 560)
	downloads, ok := tt.Find(locales.SidebarDownloads())
	if !ok {
		t.Fatal("downloads entry is missing")
	}
	license, ok := tt.Find(locales.SidebarLicenseLabel() + ":")
	if !ok {
		t.Fatal("license caption is missing")
	}
	tt.Scroll(float32(downloads.X+downloads.W/2), float32(license.Y-40), 0, 700)
	tt.Frame()
	after, ok := tt.Find(locales.SidebarDownloads())
	if !ok || after.Y != downloads.Y || after.Y+after.H > 560 {
		t.Fatalf("scrolling the navigation moved downloads: before=%+v after=%+v", downloads, after)
	}
	if downloads.Y < license.Y+license.H {
		t.Fatalf("downloads sits above the license caption: downloads=%+v license=%+v", downloads, license)
	}
}
