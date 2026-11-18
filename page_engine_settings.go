package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"unitystarter/locales"
)

// engineSettingsPage holds settings for the active engine, while the pinned
// System settings page holds launcher-wide settings. The two are deliberately
// separate so importing an engine never mixes with language, updates or
// global project roots.
func (a *app) engineSettingsPage(c *ui.Context) {
	t := c.Theme()
	s := a.engineStrategy()
	adapter := s.Adapter
	name := s.Name
	a.header(c, locales.EngineSettings(), locales.EngineSettingsHint(name), nil)
	ui.Divider(c)
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(24).Gap(16).FillWidth().Children(func() {
			if adapter == nil {
				a.emptyState(c, locales.EngineSettings(), locales.EngineNoExtraSettings())
				return
			}
			if !isExternalEngine(adapter.ID) {
				panel(c, func() {
					a.sectionTitle(c, locales.EngineInstallRoot(), locales.EngineInstallRootHint(name))
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, a.installRoot()).FontSize(12).TextColor(t.TextMuted).Grow(1).MinWidth(0).
							SingleLine().Clip().Tooltip(a.installRoot())
						ui.Button(c, locales.InstallChooseLocation()).OnClick(a.chooseInstallRoot)
						ui.Button(c, locales.ActionAddFolder()).OnClick(a.addEditorRoot)
					})
				})
			}
			if adapter.Features.Downloads {
				panel(c, func() {
					a.sectionTitle(c, locales.EngineDownloadWorkers(), locales.EngineDownloadWorkersHint(name))
					ui.Row(c).Gap(6).Children(func() {
						current := a.downloadWorkers(adapter.ID)
						for _, n := range []int{1, 2, 4, 6} {
							n := n
							pageTab(c, fmt.Sprint(n), current == n, func() {
								a.setDownloadWorkers(adapter.ID, n)
							})
						}
					})
				})
			}
			if adapter.ID == kindUnity {
				panel(c, func() {
					a.sectionTitle(c, locales.PageLicense(), locales.PageLicenseHint())
					ui.Row(c).Gap(8).Wrap().Children(func() {
						ui.Button(c, locales.EngineManageLicense()).OnClick(func() { a.page = pageLicense })
					})
				})
			}
			panel(c, func() {
				a.sectionTitle(c, locales.EngineAdapterInfo(), locales.EngineAdapterInfoHint())
				rows := [][2]string{
					{"ID", adapter.ID},
					{strings.TrimSpace(locales.SettingsVersion("")), "v" + adapter.Version},
					{locales.EngineAdapterInfo(), locales.EnginesPages(fmt.Sprint(len(adapter.Pages)))},
				}
				for _, row := range rows {
					ui.Row(c).Gap(12).Children(func() {
						ui.Text(c, row[0]).FontSize(11).TextColor(t.TextMuted).Width(140)
						ui.Text(c, row[1]).FontSize(12).Grow(1).MinWidth(0).SingleLine().Clip().Tooltip(row[1])
					})
				}
				if adapter.Homepage != "" {
					ui.Text(c, adapter.Homepage).FontSize(11).TextColor(t.TextMuted).SingleLine().Clip().Tooltip(adapter.Homepage)
				}
				ui.Row(c).Gap(8).Wrap().Children(func() {
					if adapter.URLs.Docs != "" {
						ui.Button(c, locales.EngineOpenDocs()).OnClick(func() { _ = mygo.Shell.OpenExternal(adapter.URLs.Docs) })
					}
					if adapter.URLs.Community != "" {
						ui.Button(c, locales.LinkCommunity()).OnClick(func() { _ = mygo.Shell.OpenExternal(adapter.URLs.Community) })
					}
					if adapter.Dir != "" {
						ui.Button(c, locales.ActionFolder()).OnClick(a.openEngineAdapterFolder)
					}
					if !adapter.BuiltIn {
						ui.Button(c, locales.EnginesRemove()).OnClick(func() { a.removeEngineAdapter(adapter.ID) })
					}
				})
			})
		})
	})
}
