package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"unitystarter/locales"
)

// externalPage renders a page owned by an imported adapter. Built-in kinds
// reuse the existing native workspaces; everything else is a manifest-defined
// links page, so an adapter can add a destination without changing this app.
func (a *app) externalPage(c *ui.Context, adapter *engineAdapter, page enginePage) {
	switch page.Kind {
	case "projects", "favorites":
		a.projectsPage(c)
	case "installs":
		a.installsPage(c)
	case "activity":
		a.activityPage(c)
	case "downloads":
		a.emptyState(c, page.TitleText(), locales.EnginesNoLinks())
	default:
		a.adapterLinksPage(c, adapter, page)
	}
}

func (a *app) adapterLinksPage(c *ui.Context, adapter *engineAdapter, page enginePage) {
	hint := strings.TrimSpace(adapter.Description)
	if hint == "" {
		hint = adapter.DisplayName()
	}
	a.header(c, page.TitleText(), hint, func() {
		// Imported adapters take their homepage from adapter.json. Built-in
		// adapter homes are storefronts or documentation, not GitHub repos, so
		// the GitHub action would mislabel them.
		if adapter.Homepage != "" && !adapter.BuiltIn {
			ui.Button(c, "GitHub ↗").Label(locales.OpenExternal(adapter.DisplayName())).OnClick(func() {
				_ = mygo.Shell.OpenExternal(adapter.Homepage)
			})
		}
	})
	ui.Divider(c)
	groups := append([]engineLinkGroup(nil), page.Groups...)
	if page.URL != "" {
		groups = append(groups, engineLinkGroup{
			Title: page.TitleText(),
			Links: []engineLink{{Title: page.TitleText(), Detail: adapter.DisplayName(), URL: page.URL}},
		})
	}
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(24).Gap(16).FillWidth().Children(func() {
			if len(groups) == 0 {
				a.emptyState(c, page.TitleText(), locales.EnginesNoLinks())
				return
			}
			for _, group := range groups {
				group := group
				panel(c, func() {
					ui.Text(c, group.Title).FontSize(14).Bold()
					for i, l := range group.Links {
						l := l
						if i > 0 {
							ui.Divider(c)
						}
						c.Key(fmt.Sprintf("%s#%d", l.URL, i))
						ui.Row(c).Gap(14).AlignItems(ui.Center).Padding(6, 0).Children(func() {
							ui.Box(c).Size(36, 36).Shrink(0).Radius(8).Background(c.Theme().SurfaceHover).Border(1, c.Theme().Border).Center().Children(func() {
								ui.Icon(c, page.Icon).Size(20, 20).TextColor(c.Theme().TextMuted)
							})
							ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
								ui.Text(c, l.Title).FontSize(13).Bold()
								if l.Detail != "" {
									ui.Text(c, l.Detail).FontSize(11).TextColor(c.Theme().TextMuted).SingleLine().Clip().Tooltip(l.Detail)
								}
							})
							if ui.Button(c, locales.ActionOpen()+" ↗").Label(locales.OpenExternal(l.Title)).Tooltip(l.URL).Shrink(0).Clicked() {
								_ = mygo.Shell.OpenExternal(l.URL)
							}
						})
					}
				})
			}
		})
	})
}

func (a *app) enginesSection(c *ui.Context) {
	t := c.Theme()
	panel(c, func() {
		a.sectionTitle(c, locales.EnginesTitle(), locales.EnginesHint())
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, &a.adapterURL).Placeholder(locales.EnginesImportPlaceholder()).
				Label(locales.EnginesImportPlaceholder()).Grow(1).Disabled(a.adapterBusy)
			ui.PrimaryButton(c, locales.EnginesImport()).Disabled(a.adapterBusy || strings.TrimSpace(a.adapterURL) == "").
				OnClick(a.startEngineAdapterImport)
		})
		if a.adapterBusy {
			ui.Progress(c, -1)
		}
		if a.adapterStatus != "" {
			color := t.TextMuted
			if a.adapterStatusIsError {
				color = t.Danger
			}
			ui.Text(c, a.adapterStatus).FontSize(11).TextColor(color).SingleLine().Clip().Tooltip(a.adapterStatus)
		}
		for _, err := range a.adapterErrors {
			ui.Text(c, err.Error()).FontSize(11).TextColor(t.Danger).Clip().Tooltip(err.Error())
		}
		ui.Row(c).Gap(8).Wrap().Children(func() {
			ui.Button(c, locales.EnginesOpenFolder()).OnClick(a.openEngineAdapterFolder)
			ui.Button(c, locales.EnginesSpec()).OnClick(func() { _ = mygo.Shell.OpenExternal(engineAdapterSpecURL()) })
		})
	})

	panel(c, func() {
		a.sectionTitle(c, locales.EnginesInstalled(), locales.EnginesInstalledHint())
		external := 0
		for _, adapter := range engineAdapters() {
			if !adapter.BuiltIn {
				external++
			}
			a.engineAdapterRow(c, adapter)
		}
		if external == 0 {
			ui.Text(c, locales.EnginesEmpty()).FontSize(12).TextColor(t.TextMuted)
		}
	})
}

func (a *app) engineAdapterRow(c *ui.Context, adapter *engineAdapter) {
	t := c.Theme()
	ui.Row(c).Gap(12).AlignItems(ui.Center).Padding(6, 0).Children(func() {
		engineMark(c, adapter.ID, 36, 22)
		ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, adapter.DisplayName()).FontSize(13).Bold().SingleLine().Clip()
				ui.Text(c, "v"+adapter.Version).FontSize(10).TextColor(t.TextMuted).SingleLine()
				kind := locales.EnginesExternal()
				if adapter.BuiltIn {
					kind = locales.EnginesBuiltin()
				}
				ui.Badge(c, kind)
				if adapter.Process != nil {
					ui.Badge(c, locales.EnginesProcess())
				}
			})
			if adapter.Description != "" {
				ui.Text(c, adapter.Description).FontSize(11).TextColor(t.TextMuted).SingleLine().Clip().Tooltip(adapter.Description)
			}
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, locales.EnginesPages(fmt.Sprint(len(adapter.Pages)))).FontSize(10).TextColor(t.TextMuted)
				if adapter.Source != "" {
					ui.Text(c, adapter.Source).FontSize(10).TextColor(t.TextMuted).Grow(1).MinWidth(0).SingleLine().Clip().Tooltip(adapter.Source)
				}
			})
		})
		if !adapter.BuiltIn {
			ui.Button(c, locales.EnginesRemove()).OnClick(func() { a.removeEngineAdapter(adapter.ID) })
		}
	})
}

func (a *app) startEngineAdapterImport() {
	if a.adapterBusy {
		return
	}
	raw := strings.TrimSpace(a.adapterURL)
	if raw == "" {
		return
	}
	a.adapterBusy = true
	a.adapterStatusIsError = false
	a.adapterStatus = locales.EnginesImporting()
	win := a.win
	if win == nil {
		a.adapterBusy = false
		a.adapterStatusIsError = true
		a.adapterStatus = locales.EnginesImportFailed("window unavailable")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		adapter, err := importEngineAdapter(ctx, &http.Client{Timeout: 20 * time.Second}, raw, adaptersDir())
		var errs []error
		if err == nil {
			errs = reloadEngineAdapters(adaptersDir())
		}
		win.Update(func() {
			a.adapterBusy = false
			a.adapterErrors = errs
			if err != nil {
				a.adapterStatusIsError = true
				a.adapterStatus = locales.EnginesImportFailed(err.Error())
				return
			}
			a.adapterStatusIsError = false
			a.adapterStatus = locales.EnginesImported(adapter.DisplayName())
			a.adapterURL = ""
			if engineValid(adapter.ID) {
				a.activeEngine = adapter.ID
				a.page = engineStrategyFor(adapter.ID).Pages.Projects
				a.prefs.ActiveEngine = adapter.ID
				a.save()
				a.refresh()
			}
		})
	}()
}

func (a *app) removeEngineAdapter(id string) {
	name := engineDisplayName(id)
	if err := removeInstalledEngineAdapter(adaptersDir(), id); err != nil {
		a.adapterStatusIsError = true
		a.adapterStatus = locales.EnginesImportFailed(err.Error())
		return
	}
	a.adapterErrors = reloadEngineAdapters(adaptersDir())
	a.adapterStatusIsError = false
	a.adapterStatus = locales.EnginesRemoved(name)
	if a.activeEngine == id {
		a.switchEngine(kindUnity)
	}
	a.refresh()
}

func (a *app) openEngineAdapterFolder() {
	dir := adaptersDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.alert(locales.OpenFailed(err.Error()))
		return
	}
	if err := mygo.Shell.OpenPath(dir); err != nil {
		a.alert(locales.OpenFailed(err.Error()))
	}
}
