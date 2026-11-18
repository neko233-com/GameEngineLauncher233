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

func TestEngineRailKeyboardShortcuts(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Key(ui.Ctrl, ui.Key2)
	if a.activeEngine != kindTuanjie || a.page != pageTuanjieProjects {
		t.Fatalf("Ctrl+2 left engine=%q page=%q", a.activeEngine, a.page)
	}
	tt.Key(ui.Ctrl, ui.Key1)
	if a.activeEngine != kindUnity || a.page != pageProjects {
		t.Fatalf("Ctrl+1 left engine=%q page=%q", a.activeEngine, a.page)
	}
	tt.Key(ui.Ctrl, ui.KeyComma)
	if a.page != pageSettings {
		t.Fatalf("Ctrl+, opened %q", a.page)
	}
	a.page = pageProjects
	a.alertOpen, a.alertMessage = true, "stay in this dialog"
	tt.Frame()
	tt.Key(ui.Ctrl, ui.Key2)
	if a.activeEngine != kindUnity {
		t.Fatalf("shortcut switched engines behind a dialog: %q", a.activeEngine)
	}
}

// The rail is one Tab stop and the arrow keys walk it, as a Vercel tab list.
func TestEngineRailArrowKeysSwitchFocus(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if a.activeEngine != kindTuanjie {
		t.Fatalf("down arrow did not reach the Tuanjie tab: engine=%q", a.activeEngine)
	}
	tt.Key(0, ui.KeyUp)
	tt.Key(0, ui.KeyEnter)
	if a.activeEngine != kindUnity {
		t.Fatalf("up arrow did not return to Unity: engine=%q", a.activeEngine)
	}
}

func TestNavigationFocusGroupWalksItems(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	if err := tt.Click(locales.SidebarProjects()); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if a.page != pageInstalls {
		t.Fatalf("down arrow on navigation opened %q", a.page)
	}
}

// The Vercel-style palette keeps the canvas black, cards near black and the
// primary action white.
func TestVercelMonochromePalette(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	r, ok := tt.Find(locales.ActionNewProject())
	if !ok {
		t.Fatal("New project button is missing")
	}
	img := tt.Image()
	button := img.RGBAAt(int(r.X-8), int(r.Y+r.H/2))
	if button.R < 220 || button.G < 220 || button.B < 220 {
		t.Fatalf("primary action is not white: %+v", button)
	}
	canvas := img.RGBAAt(700, 700)
	if canvas.R > 8 || canvas.G > 8 || canvas.B > 8 {
		t.Fatalf("canvas is not black: %+v", canvas)
	}
	nav := img.RGBAAt(300, 620)
	if nav.R > 16 || nav.G > 16 || nav.B > 16 {
		t.Fatalf("navigation surface is not near black: %+v", nav)
	}
}
