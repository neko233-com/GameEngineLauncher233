package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"unitystarter/locales"
)

// globalSettingsPage is the launcher-wide engine overview pinned at the bottom
// of the navigation list. It mirrors every engine's install root and download
// concurrency without switching the active engine first.
func (a *app) globalSettingsPage(c *ui.Context) {
	a.header(c, locales.SidebarGlobalSettings(), locales.GlobalSettingsHint(), func() {
		ui.Button(c, locales.SidebarSettings()).OnClick(func() {
			a.settingsTab = ""
			a.page = pageSettings
		})
	})
	ui.Divider(c)
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(24).Gap(16).FillWidth().Children(func() {
			panel(c, func() {
				a.sectionTitle(c, locales.GlobalSettingsShared(), locales.GlobalSettingsSharedHint())
				ui.Row(c).Gap(8).Wrap().Children(func() {
					ui.Button(c, locales.SettingsProjectRoots()).OnClick(func() {
						a.settingsTab = "projects"
						a.page = pageSettings
					})
					ui.Button(c, locales.SettingsEditorRoots()).OnClick(func() {
						a.settingsTab = "installs"
						a.page = pageSettings
					})
					ui.Button(c, locales.EnginesTitle()).OnClick(func() {
						a.settingsTab = "engines"
						a.page = pageSettings
					})
				})
			})
			panel(c, func() {
				a.sectionTitle(c, locales.GlobalSettingsAllEngines(), locales.GlobalSettingsEnginesHint())
				for i, adapter := range engineAdapters() {
					if i > 0 {
						ui.Divider(c)
					}
					a.globalEngineRow(c, adapter)
				}
			})
		})
	})
}

func (a *app) globalEngineRow(c *ui.Context, adapter *engineAdapter) {
	t := c.Theme()
	c.Key(adapter.ID)
	ui.Column(c).Padding(2, 0).Gap(8).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			engineMark(c, adapter.ID, 34, 22)
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, adapter.DisplayName()).FontSize(14).Bold().SingleLine().Clip().Tooltip(adapter.DisplayName())
					if adapter.BuiltIn {
						ui.Badge(c, locales.EnginesBuiltin())
					} else {
						ui.Badge(c, locales.EnginesExternal())
					}
				})
				detail := strings.TrimSpace(adapter.Description)
				if detail == "" {
					detail = adapter.ID
				}
				ui.Text(c, detail).FontSize(11).TextColor(t.TextMuted).SingleLine().Clip()
			})
			ui.Button(c, locales.EngineSettings()).OnClick(func() {
				if adapter.ID != a.activeEngine {
					a.switchEngine(adapter.ID)
				}
				a.page = pageEngineSettings
			})
		})
		if !isExternalEngine(adapter.ID) {
			root := a.installRootFor(adapter.ID)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, locales.EngineInstallRoot()).FontSize(11).TextColor(t.TextMuted).Width(110)
				ui.Text(c, root).Grow(1).MinWidth(0).FontSize(12).TextColor(t.TextMuted).SingleLine().Clip().Tooltip(root)
				ui.Button(c, locales.InstallChooseLocation()).OnClick(func() {
					a.chooseInstallRootFor(adapter.ID)
				})
				ui.Button(c, locales.ActionAddFolder()).OnClick(a.addEditorRoot)
			})
		}
		if adapter.Features.Downloads {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, locales.EngineDownloadWorkers()).FontSize(11).TextColor(t.TextMuted).Width(110)
				current := a.downloadWorkers(adapter.ID)
				for _, n := range []int{1, 2, 4, 6} {
					n := n
					pageTab(c, fmt.Sprint(n), current == n, func() {
						a.setDownloadWorkers(adapter.ID, n)
					})
				}
			})
		}
	})
}
