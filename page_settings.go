package main

import (
	"os"
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"unitystarter/locales"
)

func (a *app) settingsPage(c *ui.Context) {
	a.header(c, locales.SettingsTitle(), locales.SettingsHint(), nil)
	ui.Row(c).Padding(0, 24).Gap(8).Children(func() {
		for _, tab := range []struct{ id, label string }{
			{"", locales.SettingsGeneral()}, {"projects", locales.SettingsProjectsTab()},
			{"installs", locales.SettingsInstallsTab()}, {"engines", locales.SettingsEnginesTab()},
			{"about", locales.SettingsAbout()},
		} {
			tab := tab
			pageTab(c, tab.label, a.settingsTab == tab.id, func() { a.settingsTab = tab.id })
		}
	})
	ui.Divider(c)
	c.Key(a.settingsTab)
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(24).Gap(16).FillWidth().Children(func() {
			switch a.settingsTab {
			case "projects":
				a.hiddenProjectsSection(c)
				a.rootsSection(c, locales.SettingsProjectRoots(), locales.SettingsProjectRootsHint(), a.prefs.ProjectRoots,
					func(next []string) {
						a.prefs.ProjectRoots = next
						a.save()
						a.refresh()
					},
					func() {
						a.pickFolder(locales.DialogChooseFolder(), func(dir string) {
							a.prefs.ProjectRoots = dedupe(append(a.prefs.ProjectRoots, dir))
							a.save()
							a.refresh()
						})
					})

				a.rootsSection(c, locales.SettingsPinnedProjects(), locales.SettingsPinnedProjectsHint(), a.prefs.PinnedProjects,
					func(next []string) {
						a.prefs.PinnedProjects = next
						a.save()
						a.refresh()
					}, a.addProject)
			case "installs":
				panel(c, func() {
					a.sectionTitle(c, locales.EngineSettings(), locales.EngineInstallRootHint(channelName(a.activeEngine)))
					ui.Button(c, locales.EngineSettings()).OnClick(func() { a.page = pageEngineSettings })
				})
				a.rootsSection(c, locales.SettingsEditorRoots(), locales.SettingsEditorRootsHint(), a.prefs.EditorRoots,
					func(next []string) {
						a.prefs.EditorRoots = next
						a.save()
						a.refresh()
					},
					func() {
						a.pickFolder(locales.DialogChooseFolder(), func(dir string) {
							a.prefs.EditorRoots = dedupe(append(a.prefs.EditorRoots, dir))
							a.save()
							a.refresh()
						})
					})

				a.downloadCacheSection(c)
			case "engines":
				a.enginesSection(c)
			case "about":
				a.aboutSection(c)
			default:
				a.languageSection(c)
				a.hubSection(c)
				a.demoSection(c)
			}
		})
	})
}

func (a *app) hiddenProjectsSection(c *ui.Context) {
	if len(a.prefs.HiddenProjects) == 0 {
		return
	}
	paths := make([]string, 0, len(a.prefs.HiddenProjects))
	for _, path := range a.prefs.HiddenProjects {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	panel(c, func() {
		a.sectionTitle(c, locales.HiddenProjectsTitle(), locales.HiddenProjectsHint())
		for _, path := range paths {
			path := path
			c.Key(path)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, path).Grow(1).MinWidth(0).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine().Clip().Tooltip(path)
				ui.Button(c, locales.ProjectRestore()).OnClick(func() { a.setProjectHidden(path, false) })
			})
		}
	})
}

func (a *app) downloadCacheSection(c *ui.Context) {
	t := c.Theme()
	dir := downloadsDir()
	panel(c, func() {
		a.sectionTitle(c, locales.SettingsDownloadCache(), locales.SettingsDownloadCacheHint())
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, dir).FontSize(12).TextColor(t.TextMuted).Grow(1).SingleLine().Clip().Tooltip(dir)
			ui.Button(c, locales.ActionFolder()).OnClick(func() {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					a.alert(locales.OpenFailed(err.Error()))
					return
				}
				if err := mygo.Shell.OpenPath(dir); err != nil {
					a.alert(locales.OpenFailed(err.Error()))
				}
			})
		})
	})
}

func (a *app) sectionTitle(c *ui.Context, title, hint string) {
	t := c.Theme()
	ui.Column(c).Gap(2).Children(func() {
		ui.Text(c, title).FontSize(15).Bold()
		ui.Text(c, hint).FontSize(12).TextColor(t.TextMuted)
	})
}

func (a *app) languageSection(c *ui.Context) {
	labels := []string{locales.SettingsLanguageSystem(), "English", "简体中文"}
	tags := []string{"", "en", "zh-Hans"}
	index := 0
	switch {
	case a.prefs.Locale == "":
		index = 0
	case strings.HasPrefix(a.prefs.Locale, "zh"):
		index = 2
	default:
		index = 1
	}
	chosen := labels[index]

	panel(c, func() {
		a.sectionTitle(c, locales.SettingsLanguage(), locales.SettingsLanguageHint())
		ui.Select(c, &chosen, labels).Width(220)
	})

	for i, label := range labels {
		if label == chosen && i != index {
			a.prefs.Locale = tags[i]
			a.save()
			mygo.App.SetLocale(a.prefs.Locale)
		}
	}
}

// rootsSection lists folders, with a button to add one and one to remove
// each.
func (a *app) rootsSection(c *ui.Context, title, hint string, roots []string, set func([]string), add func()) {
	t := c.Theme()
	c.Key(title)
	panel(c, func() {
		a.sectionTitle(c, title, hint)
		if len(roots) == 0 {
			ui.Text(c, locales.SettingsRootsEmpty()).FontSize(12).TextColor(t.TextMuted)
		}
		for i, root := range roots {
			i, root := i, root
			c.Key(root)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, root).FontSize(12).Grow(1).MinWidth(0).SingleLine().Clip().Tooltip(root)
				if ui.Button(c, locales.ActionRemove()).Clicked() {
					next := append([]string{}, roots[:i]...)
					next = append(next, roots[i+1:]...)
					set(next)
				}
			})
		}
		ui.Row(c).Children(func() {
			ui.Button(c, locales.ActionAddFolder()).OnClick(add).Width(140)
		})
	})
}

// chooseInstallRoot changes the active engine's editor install folder.
func (a *app) chooseInstallRoot() {
	a.pickFolder(locales.DialogChooseFolder(), func(dir string) {
		a.setInstallRoot(dir)
		a.save()
		a.refresh()
	})
}

func (a *app) setInstallRoot(dir string) {
	a.setInstallRootFor(a.activeEngine, dir)
}

// demoSection turns installs into stand-ins, for trying the flow.
func (a *app) demoSection(c *ui.Context) {
	demo := a.prefs.DemoMode
	panel(c, func() {
		a.sectionTitle(c, locales.SettingsDemo(), locales.SettingsDemoHint())
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Switch(c, &demo)
			ui.Text(c, locales.SettingsDemoSwitch()).FontSize(12)
		})
	})
	if demo != a.prefs.DemoMode {
		a.prefs.DemoMode = demo
		a.save()
	}
}

// hubSection takes over the hubs' projects and favorites, and says which of
// them this app can write to. Unity and Tuanjie are always listed so a missing
// Hub still has its official download entry.
func (a *app) hubSection(c *ui.Context) {
	t := c.Theme()
	sync := !a.prefs.NoHubSync
	panel(c, func() {
		a.sectionTitle(c, locales.SettingsHubs(), locales.SettingsHubsHint())
		for _, kind := range []string{kindUnity, kindTuanjie} {
			kind := kind
			h, ok := a.hubForEngine(kind)
			c.Key("hub-" + kind)
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, hubName(kind)).FontSize(13).Bold()
						if ok {
							ui.Badge(c, hubMode(h))
							if h.Version != "" {
								ui.Text(c, h.Version).FontSize(11).TextColor(t.TextMuted)
							} else {
								ui.Badge(c, locales.HubDetected())
							}
						} else {
							ui.Badge(c, locales.HubNotDetected())
						}
					})
					if ok {
						ui.Text(c, h.Dir).FontSize(11).TextColor(t.TextMuted).SingleLine().Clip().Tooltip(h.Dir)
					} else {
						ui.Text(c, locales.HubNotDetectedSettingsHint()).FontSize(11).TextColor(t.TextMuted)
					}
				})
				if ok {
					if exe := strings.TrimSpace(h.Exe); exe != "" {
						ui.Button(c, locales.ActionOpenHub()).OnClick(func() {
							_ = mygo.Shell.OpenPath(exe)
						})
					}
					if dir := strings.TrimSpace(h.Dir); dir != "" {
						ui.Button(c, locales.ActionOpenConfig()).OnClick(func() {
							_ = mygo.Shell.OpenPath(dir)
						})
					}
				} else {
					url := "https://unity.com/download"
					label := locales.UnityHubDownload()
					if kind == kindTuanjie {
						url = "https://unity.cn/tuanjie/releases"
						label = locales.TuanjieHubDownload()
					}
					ui.Button(c, label).OnClick(func() {
						_ = mygo.Shell.OpenExternal(url)
					})
				}
			})
			if kind == kindUnity {
				ui.Divider(c)
			}
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Switch(c, &sync)
			ui.Text(c, locales.SettingsHubSync()).FontSize(12)
		})
		ui.Text(c, locales.SettingsHubSyncHint()).FontSize(11).TextColor(t.TextMuted)
	})
	if sync == a.prefs.NoHubSync {
		a.prefs.NoHubSync = !sync
		a.save()
	}
}

func hubName(kind string) string {
	if kind == kindTuanjie {
		return locales.HubTuanjie()
	}
	return locales.HubUnity()
}

func hubMode(h hub) string {
	if h.writable() {
		return locales.HubModeBoth()
	}
	return locales.HubModeImport()
}

// hubModeCode names the mode for scripts, whatever the app's language.
func hubModeCode(h hub) string {
	if h.writable() {
		return "two-way"
	}
	return "import-only"
}

func (a *app) aboutSection(c *ui.Context) {
	t := c.Theme()
	version := mygo.App.Version()
	if version == "" {
		version = appVersion
	}
	panel(c, func() {
		a.sectionTitle(c, locales.SettingsAbout(), locales.SettingsAboutText())
		ui.Text(c, locales.SettingsVersion(version)).FontSize(12).TextColor(t.TextMuted)
		ui.Text(c, prefsPath()).FontSize(11).TextColor(t.TextMuted).SingleLine().Clip().Tooltip(prefsPath())
		if a.updateStatus != "" {
			ui.Text(c, a.updateStatus).FontSize(12).TextColor(t.TextMuted)
		}
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if ui.Button(c, locales.ActionOpenDataDir()).Clicked() {
				if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
					_ = mygo.Shell.OpenPath(dir)
				}
			}
			if ui.Button(c, locales.ActionOpenLogs()).Clicked() {
				_ = mygo.Shell.OpenPath(logPath())
			}
			if ui.PrimaryButton(c, locales.ActionCheckUpdate()).Disabled(a.updateBusy).Clicked() {
				a.checkUpdates()
			}
		})
	})
}
