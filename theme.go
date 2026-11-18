package main

import "github.com/egoist/mygo/ui"

// A Vercel-style monochrome workspace: black canvas, near-black cards, one
// hairline border tone, white primary actions and a muted gray hierarchy.
var starterTheme = func() *ui.Theme {
	t := ui.DarkTheme()
	t.Background = ui.Hex("#000000")
	t.Surface = ui.Hex("#0a0a0a")
	t.SurfaceHover = ui.Hex("#161616")
	t.SurfacePressed = ui.Hex("#1f1f1f")
	t.Border = ui.Hex("#2a2a2a")
	t.Text = ui.Hex("#ededed")
	t.TextMuted = ui.Hex("#a1a1aa")
	t.Accent = ui.Hex("#ffffff")
	t.AccentHover = ui.Hex("#d4d4d4")
	t.AccentPressed = ui.Hex("#bdbdbd")
	t.AccentText = ui.Hex("#050505")
	t.Danger = ui.Hex("#ff5c5c")
	t.Warning = ui.Hex("#f5a623")
	t.Success = ui.Hex("#3ecf8e")
	t.Selection = ui.RGBA(255, 255, 255, 0.22)
	t.Focus = ui.RGBA(255, 255, 255, 0.55)
	t.Scrollbar = ui.RGBA(255, 255, 255, 0.22)
	t.Radius = 8
	return t
}()
