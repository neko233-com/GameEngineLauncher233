package main

import (
	"github.com/egoist/mygo/ui"
	"unitystarter/locales"
)

// Shared native controls keep focus, keyboard and accessibility behavior while
// giving every workspace the same restrained Hub-style hierarchy.
func pageTab(c *ui.Context, label string, selected bool, choose func()) {
	t := c.Theme()
	c.Key(label)
	b := ui.Button(c, label).Height(30).Padding(0, 12).Radius(6).FontSize(13).
		Background(ui.Color{}).Border(1, ui.Color{}).TextColor(t.TextMuted).OnClick(choose)
	if selected {
		b.Background(t.SurfaceHover).Border(1, t.Border).TextColor(t.Text)
	}
	if b.Hovered() {
		b.Background(t.SurfaceHover)
		b.TextColor(t.Text)
	}
}

func panel(c *ui.Context, children func()) {
	t := c.Theme()
	ui.Column(c).Padding(18).Gap(12).Background(t.Surface).Border(1, t.Border).Radius(8).Children(children)
}

// modal is the app's Vercel-style dialog shell: a deep backdrop over the
// black canvas and a bordered surface card, since a black panel would
// otherwise disappear into the page. It returns the dialog for Dismissed.
func modal(c *ui.Context, open *bool, children func()) ui.Element {
	t := c.Theme()
	w, h := c.Size()
	return ui.DialogBase(c, open, func(backdrop, dialog ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.72))
		dialog.Padding(20).Gap(14).Radius(12).Background(t.Surface).Border(1, t.Border).
			MaxWidth(max(320, w-40)).MaxHeight(max(200, h-40)).
			Shadow(0, 24, 60, 0, ui.RGBA(0, 0, 0, 0.55))
		children()
	})
}

func tableHeading(c *ui.Context, label, sort string, active bool, reverse bool, choose func()) ui.Element {
	t := c.Theme()
	text := label
	if active {
		descending := !reverse
		if sort == "name" {
			descending = reverse
		}
		if descending {
			text += " ↓"
		} else {
			text += " ↑"
		}
	}
	c.Key(sort)
	b := ui.Button(c, text).Label(label).Tooltip(label).Height(30).Padding(0, 0).
		Background(ui.Color{}).Border(0, ui.Color{}).FontSize(12).TextColor(t.TextMuted).OnClick(choose)
	if sort != "favorite" {
		b.Justify(ui.Start)
	}
	if active {
		b.TextColor(t.Text)
	}
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	return b
}

func (a *app) searchEmpty(c *ui.Context, clear func()) {
	ui.Column(c).Grow(1).Center().Padding(24).Gap(12).Children(func() {
		ui.Icon(c, iconProjects).Size(30, 30).TextColor(c.Theme().TextMuted)
		ui.Text(c, locales.SearchEmpty()).FontSize(16).Bold()
		ui.Text(c, locales.SearchEmptyHint()).FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Button(c, locales.SearchClear()).OnClick(clear)
	})
}

// licenseStatusStrip is the compact line above the pinned Downloads item.
// Tuanjie Hub uses a single muted caption ("当前许可证: 个人版") rather than
// a card, so the navigation keeps its full height for real destinations.
func (a *app) licenseStatusStrip(c *ui.Context) {
	t := c.Theme()
	s := a.engineStrategy()

	label := locales.LicenseStateUnknown()
	color := t.TextMuted
	if !s.NeedsLicenseFlow {
		label = locales.LicenseStateFree()
	} else {
		switch a.licenseCheckState {
		case "active":
			label = locales.LicenseStateActive()
			color = t.Success
		case "inactive":
			label = locales.LicenseStateInactive()
			color = t.Danger
		default:
			label = locales.LicenseStateUnknown()
		}
	}

	ui.Row(c).Shrink(0).Padding(12, 14, 10, 14).Gap(4).AlignItems(ui.Center).Children(func() {
		ui.Text(c, locales.SidebarLicenseLabel()+":").FontSize(11).TextColor(t.TextMuted).SingleLine()
		ui.Text(c, label).FontSize(11).TextColor(color).SingleLine().Grow(1).MinWidth(0).Clip().Tooltip(label)
	})
}
