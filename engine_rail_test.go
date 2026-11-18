package main

import (
	"fmt"
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
	"unitystarter/locales"
)

// The engine switcher is a vertical rail at the far left, as Tuanjie Hub's:
// every engine stacks with its brand mark and name, the active one is raised,
// and the navigation column begins to its right.
func TestEngineRailIsVerticalAndSwitches(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)

	if !tt.HasText("Unity") || !tt.HasText("Tuanjie") || !tt.HasText("Godot") || !tt.HasText("Unreal") {
		t.Fatalf("rail is missing engine names: %q", tt.Texts())
	}
	unity, ok := tt.Find("Unity")
	if !ok {
		t.Fatal("the Unity tab is missing from the rail")
	}
	tuanjie, ok := tt.Find("Tuanjie")
	if !ok {
		t.Fatal("the Tuanjie tab is missing from the rail")
	}
	unreal, ok := tt.Find("Unreal")
	if !ok {
		t.Fatal("the Unreal tab is missing from the rail")
	}
	railWidth := float32(engineRailWidth)
	if unity.X < 0 || unity.X+unity.W > railWidth || unity.W < 120 || unity.H < 40 {
		t.Fatalf("Unity tab is not a wide rail tab: %+v", unity)
	}
	if tuanjie.X < 0 || tuanjie.X+tuanjie.W > railWidth {
		t.Fatalf("Tuanjie tab escapes the rail: %+v", tuanjie)
	}
	if unreal.X < 0 || unreal.X+unreal.W > railWidth {
		t.Fatalf("Unreal tab escapes the rail: %+v", unreal)
	}
	if tuanjie.Y < unity.Y+unity.H {
		t.Fatalf("engine tabs are not stacked vertically: unity=%+v tuanjie=%+v", unity, tuanjie)
	}
	if unreal.Y < tuanjie.Y+tuanjie.H {
		t.Fatalf("Unreal is not stacked below Tuanjie: tuanjie=%+v unreal=%+v", tuanjie, unreal)
	}

	projects, ok := tt.Find(locales.SidebarProjects())
	if !ok || projects.X < railWidth {
		t.Fatalf("navigation overlaps the engine rail: %+v, found=%v", projects, ok)
	}
	gear, ok := tt.Find(locales.EngineSettings())
	if !ok || gear.Y+gear.H > 120 {
		t.Fatalf("engine settings gear is not in the navigation header: %+v, found=%v", gear, ok)
	}
	system, ok := tt.Find(locales.SidebarSettings())
	if !ok || system.Y < 600 {
		t.Fatalf("system settings is not pinned at the bottom: %+v, found=%v", system, ok)
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
	// background. Sampling the lower-left corner avoids the icon and label.
	img := tt.Image()
	active := img.RGBAAt(int(unity.X)+4, int(unity.Y+unity.H)-6)
	idle := img.RGBAAt(int(tuanjie.X)+4, int(tuanjie.Y+tuanjie.H)-6)
	if active.R <= idle.R+0x10 {
		t.Fatalf("active engine tab has no raised tile: active=%+v idle=%+v", active, idle)
	}

	if err := tt.Click("Tuanjie"); err != nil {
		t.Fatal(err)
	}
	if a.activeEngine != kindTuanjie || a.page != pageTuanjieProjects {
		t.Fatalf("rail click left engine=%q page=%q", a.activeEngine, a.page)
	}
	if !tt.HasText(locales.ChannelTuanjie()) {
		t.Fatalf("navigation header did not follow the rail: %q", tt.Texts())
	}
	if err := tt.Click("Unreal"); err != nil {
		t.Fatal(err)
	}
	if a.activeEngine != kindUnreal || a.page != pageUnrealProjects {
		t.Fatalf("Unreal rail click left engine=%q page=%q", a.activeEngine, a.page)
	}
	if !tt.HasText(locales.ChannelUnreal()) {
		t.Fatalf("navigation header did not follow the Unreal rail tab: %q", tt.Texts())
	}
}

// railTestEngineNames overflow a 900×560 window, so the rail has to scroll.
var railTestEngineNames = []string{"O3DE", "Cocos", "Defold", "Bevy", "GameMaker", "LÖVE", "RPG Maker", "CryEngine", "Flax", "Stride", "Armory", "Wave"}

func addRailTestEngines(t *testing.T) {
	t.Helper()
	withCleanAdapterRegistry(t)
	for i, name := range railTestEngineNames {
		name := name
		engineRegistryMu.Lock()
		addEngineAdapterLocked(&engineAdapter{
			ID:   fmt.Sprintf("extra%d", i),
			Icon: iconUnity,
			Pages: []enginePage{{
				Key:   fmt.Sprintf("extra%d", i),
				Route: fmt.Sprintf("/extra%d", i),
				Kind:  "links",
			}},
			name: func() string { return name },
		})
		engineRegistryMu.Unlock()
	}
}

// The rail keeps its names as adapters accumulate and scrolls instead of
// squeezing or hiding the engine list.
func TestEngineRailScrollsThroughManyEngines(t *testing.T) {
	addRailTestEngines(t)
	a := fixture()
	tt := ui.NewTester(a.view, 900, 560)

	last, ok := tt.Find("Wave")
	if !ok {
		t.Fatalf("the last engine is missing from the rail: %q", tt.Texts())
	}
	if last.H != 0 {
		t.Fatalf("the fixture did not overflow the rail: %+v", last)
	}
	tt.Scroll(engineRailWidth/2, 300, 0, 2000)
	tt.Frame()
	last, ok = tt.Find("Wave")
	if !ok || last.H < 40 || last.Y < 0 || last.Y+last.H > 560 {
		t.Fatalf("the rail did not scroll to the last engine: %+v, found=%v", last, ok)
	}
	if last.X < 0 || last.X+last.W > engineRailWidth {
		t.Fatalf("the last engine escaped the rail: %+v", last)
	}
}

// End reaches the last engine even when it starts below the viewport, and the
// rail scrolls it into view before Enter switches engines.
func TestEngineRailKeyboardFocusScrollsIntoView(t *testing.T) {
	addRailTestEngines(t)
	a := fixture()
	tt := ui.NewTester(a.view, 900, 560)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	last, ok := tt.Find("Wave")
	if !ok || last.H < 40 || last.Y < 0 || last.Y+last.H > 560 {
		t.Fatalf("End did not scroll the last engine into view: %+v, found=%v", last, ok)
	}
	tt.Key(0, ui.KeyEnter)
	if a.activeEngine != fmt.Sprintf("extra%d", len(railTestEngineNames)-1) {
		t.Fatalf("Enter did not switch to the last engine: %q", a.activeEngine)
	}
}

// A restored or shortcut-selected engine is revealed even when it starts
// below the rail viewport.
func TestEngineRailRevealsActiveEngineOnStartup(t *testing.T) {
	addRailTestEngines(t)
	a := fixture()
	lastID := fmt.Sprintf("extra%d", len(railTestEngineNames)-1)
	a.activeEngine, a.page = lastID, "/"+lastID
	tt := ui.NewTester(a.view, 900, 560)
	last, ok := tt.Find("Wave")
	if !ok || last.H < 40 || last.Y < 0 || last.Y+last.H > 560 {
		t.Fatalf("the chosen engine was not revealed: %+v, found=%v", last, ok)
	}
}

// Configuring the title bar after the first frame gives the rail a new
// viewport; the active engine must still be revealed.
func TestEngineRailRevealsActiveEngineAfterTitleBarChange(t *testing.T) {
	addRailTestEngines(t)
	a := fixture()
	lastID := fmt.Sprintf("extra%d", len(railTestEngineNames)-1)
	a.activeEngine, a.page = lastID, "/"+lastID
	tt := ui.NewTester(a.view, 900, 560)
	tt.SetTitleBar(ui.TitleBar{Height: 36, Right: 138})
	last, ok := tt.Find("Wave")
	if !ok || last.H < 40 || last.Y < 0 || last.Y+last.H > 560 {
		t.Fatalf("title bar layout hid the chosen engine: %+v, found=%v", last, ok)
	}
}

// A long community adapter name clips inside the rail instead of widening it
// or pushing the navigation column aside.
func TestEngineRailClipsLongAdapterNames(t *testing.T) {
	withCleanAdapterRegistry(t)
	const longName = "Extremely Long Community Engine Name That Must Not Widen The Rail"
	engineRegistryMu.Lock()
	addEngineAdapterLocked(&engineAdapter{
		ID:   "longname",
		Icon: iconUnity,
		Pages: []enginePage{{
			Key:   "hub",
			Route: "/longname/hub",
			Kind:  "links",
		}},
		name: func() string { return longName },
	})
	engineRegistryMu.Unlock()

	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	tab, ok := tt.Find(longName)
	if !ok {
		t.Fatalf("the long-named adapter is missing from the rail: %q", tt.Texts())
	}
	if tab.X < 0 || tab.X+tab.W > engineRailWidth || tab.W > engineRailTabWidth+1 {
		t.Fatalf("the long name widened or escaped the rail: %+v", tab)
	}
	projects, ok := tt.Find(locales.SidebarProjects())
	if !ok || projects.X < engineRailWidth {
		t.Fatalf("the navigation column moved under the rail: %+v, found=%v", projects, ok)
	}
}

// Rail tabs answer the pointer with a raised face, including the active tab.
func TestEngineRailHoverFace(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	unity, ok := tt.Find("Unity")
	if !ok {
		t.Fatal("the Unity tab is missing from the rail")
	}
	tuanjie, ok := tt.Find("Tuanjie")
	if !ok {
		t.Fatal("the Tuanjie tab is missing from the rail")
	}
	sample := func(r ui.Rect) image.Point {
		return image.Pt(int(r.X+r.W-16), int(r.Y+r.H/2))
	}
	unityAt, tuanjieAt := sample(unity), sample(tuanjie)
	img := tt.Image()
	activeBefore, idleBefore := img.RGBAAt(unityAt.X, unityAt.Y), img.RGBAAt(tuanjieAt.X, tuanjieAt.Y)

	tt.Move(unity.X+unity.W/2, unity.Y+unity.H/2)
	tt.Frame()
	tt.Frame()
	activeAfter := tt.Image().RGBAAt(unityAt.X, unityAt.Y)
	if activeAfter.R < activeBefore.R+4 || activeAfter.G < activeBefore.G+4 || activeAfter.B < activeBefore.B+4 {
		t.Fatalf("active rail tab did not light on hover: before=%+v after=%+v", activeBefore, activeAfter)
	}

	tt.Move(tuanjie.X+tuanjie.W/2, tuanjie.Y+tuanjie.H/2)
	tt.Frame()
	tt.Frame()
	idleAfter := tt.Image().RGBAAt(tuanjieAt.X, tuanjieAt.Y)
	if idleAfter.R < idleBefore.R+4 || idleAfter.G < idleBefore.G+4 || idleAfter.B < idleBefore.B+4 {
		t.Fatalf("rail tab did not light on hover: before=%+v after=%+v", idleBefore, idleAfter)
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
	tt.Key(ui.Ctrl, ui.Key4)
	if a.activeEngine != kindUnreal || a.page != pageUnrealProjects {
		t.Fatalf("Ctrl+4 left engine=%q page=%q", a.activeEngine, a.page)
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

// Navigation rows answer the pointer with the quiet Vercel hover face.
func TestNavigationHoverFace(t *testing.T) {
	a := fixture()
	tt := ui.NewTester(a.view, 1180, 760)
	r, ok := tt.Find(locales.SidebarInstalls())
	if !ok {
		t.Fatal("installs navigation row is missing")
	}
	x, y := int(r.X)+4, int(r.Y+r.H/2)
	before := tt.Image().RGBAAt(x, y)
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	tt.Frame()
	after := tt.Image().RGBAAt(x, y)
	if after.R < before.R+4 || after.G < before.G+4 || after.B < before.B+4 {
		t.Fatalf("navigation row did not light on hover: before=%+v after=%+v", before, after)
	}
}
