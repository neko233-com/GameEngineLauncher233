package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"

	"unitystarter/locales"
)

const (
	pageProjects         = "/projects"
	pageFavorites        = "/favorites"
	pageInstalls         = "/installs"
	pageDownloads        = "/downloads"
	pageTuanjieProjects  = "/tuanjie/projects"
	pageTuanjieFavorites = "/tuanjie/favorites"
	pageTuanjieInstalls  = "/tuanjie/installs"
	pageTuanjieDownloads = "/tuanjie/downloads"
	pageGodotProjects    = "/godot/projects"
	pageGodotInstalls    = "/godot/installs"
	pageGodotFavorites   = "/godot/favorites"
	pageGodotGet         = "/godot/get"
	pageGodotDocs        = "/godot/docs"
	pageGodotCommunity   = "/godot/community"
	pageStore            = "/store"
	pagePublish          = "/publish"
	pageLicense          = "/license"
	pageDocs             = "/docs"
	pageLearn            = "/learn"
	pageFeedback         = "/feedback"
	pageSettings         = "/settings"
	pageEngineSettings   = "/settings/engine"
	pageActivity         = "/activity"

	// appVersion is what the CLI prints and what the window shows when MyGo
	// does not know the packaged version.
	appVersion = "1.9.0"
	// appName is the name of the executable and of the app's data folder, as
	// mygo.json names the app; the window shows a name with a space.
	appName = "GameEngineLauncher233"
)

// app is the state of the window: what was found on this machine, the page
// shown, and the dialog being filled.
type app struct {
	win                     *mygo.Window
	page                    string
	query                   string
	queryEngine             string
	activeEngine            string
	projectSort             string
	projectSortReverse      bool
	installQuery            string
	installFilter           string
	settingsTab             string
	downloadFilterUnity     string
	downloadFilterTuanjie   string
	versionOpen             bool
	versionConfirm          bool
	versionProject          projectEntry
	versionEditor           string
	versionConfirmedEditor  editorInstall
	launchEditor            func(string, []string, string) error
	manageOpen              bool
	manageEditor            editorInstall
	installedModulesOpen    bool
	installedModulesEditor  editorInstall
	manageModules           bool
	manageUninstall         bool
	uninstallBusy           bool
	uninstallStatus         string
	uninstallGuards         []uninstallGuard
	uninstallGuardOpen      bool
	uninstallGuardConfirm   bool
	uninstallGuardSelected  uninstallGuard
	moduleBusy              bool
	moduleLoading           bool
	moduleInstalling        bool
	moduleStatus            string
	moduleSelection         map[string]bool
	moduleQueue             []string
	moduleQueueTotal        int
	moduleQueueWarnings     []string
	moduleQueueRunner       func(editorInstall, editorModule, bool)
	moduleSelectionStore    func(editorInstall, map[string]bool) error
	moduleSelectionLoad     func(editorInstall) (map[string]bool, error)
	moduleSelectionRestored bool
	moduleCurrentName       string
	moduleProgressDone      int64
	moduleProgressTotal     int64
	moduleJobSequence       uint64
	moduleDiskFree          int64
	moduleDiskKnown         bool
	licenseCheckBusy        bool
	licenseCheckState       string
	licenseStage            string
	licensePersonalConsent  bool
	licenseFlowOpen         bool
	licensePending          func()
	licensePendingLabel     string
	licenseCancel           context.CancelFunc
	licenseSequence         uint64
	licenseRunner           unityLicenseRunner
	moduleCancel            context.CancelFunc
	visualStudioTask        *visualStudioTask
	androidPlan             *androidInstallPlan
	alertOpen               bool
	alertMessage            string
	availableOpen           bool
	availableVersion        string
	availableURL            string

	editors               []editorInstall
	projects              []projectEntry
	hubs                  []hub
	prefs                 prefs
	scanning              bool
	scanAgain             bool
	scanGeneration        uint64
	scanCancel            context.CancelFunc
	projectVersionWatches map[string]*projectVersionWatch

	// hubWriteAt is when this app last wrote a hub's files, so that the
	// change the write itself reports is not read as the hub's.
	hubWriteAt time.Time
	hubCancel  context.CancelFunc

	list     ui.ListState
	installs ui.ListState

	// diagCopied says the diagnostics were just copied, beside their button.
	diagCopied bool

	// downloads is what the engines offer to install, read from the caches
	// and from Unity's releases API.
	downloads            []downloadEntry
	downloadsSince       time.Time
	downloadsBusy        bool
	downloadsReady       bool
	downloadsErrs        map[string]string
	downloadsPending     map[string]bool
	downloadQueryUnity   string
	downloadQueryTuanjie string
	unitySearchQuery     string
	unitySearchEntries   []downloadEntry
	unitySearchBusy      bool
	unitySearchErr       string
	// jobs are the editors being downloaded or installed, by engine and
	// version.
	jobs          map[string]*downloadJob
	updateBusy    bool
	updatePending string
	updateStatus  string
	updateCtx     context.Context
	updateCancel  context.CancelFunc

	newOpen                bool
	newDir                 string
	newName                string
	newEditor              string
	newEngine              string
	newError               string
	newTemplates           []projectTemplate
	newTemplateID          string
	newTemplateQuery       string
	newTemplateCategory    string
	newTemplateSource      string
	newTemplatesBusy       bool
	newTemplatesErr        string
	newTemplateChosen      bool
	newTemplateTerms       bool
	newTemplatesCancel     context.CancelFunc
	newTemplatesGeneration uint64
	newCreating            bool
	templateJob            *templateDownloadJob
	pendingProjectList     []pendingProject
	pendingProjectErrors   map[string]string

	// User-installed engine adapters and the import form.
	adapterURL           string
	adapterBusy          bool
	adapterStatus        string
	adapterStatusIsError bool
	adapterErrors        []error
}

func newApp() *app {
	return &app{page: pageProjects, activeEngine: kindUnity, launchEditor: startDetached}
}

// load reads what the last run saved, and shows the language it chose.
func (a *app) load() {
	a.prefs = loadPrefs()
	if a.prefs.Locale != "" {
		mygo.App.SetLocale(a.prefs.Locale)
	}
	a.adapterErrors = reloadEngineAdapters(adaptersDir())
	if !engineValid(a.activeEngine) {
		a.activeEngine, a.page = kindUnity, pageProjects
	}
	if a.prefs.ActiveEngine != "" && engineValid(a.prefs.ActiveEngine) {
		a.activeEngine = a.prefs.ActiveEngine
		a.page = engineStrategyFor(a.activeEngine).Pages.Projects
	}
}

// refresh scans the machine again, off the main thread.
func (a *app) refresh() {
	if a.win == nil {
		return
	}
	if a.scanning {
		a.scanAgain = true
		return
	}
	if a.scanCancel != nil {
		a.scanCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.scanCancel = cancel
	a.scanGeneration++
	generation := a.scanGeneration
	a.scanning = true
	settings := a.prefs
	settings.ProjectRoots = slices.Clone(settings.ProjectRoots)
	settings.EditorRoots = slices.Clone(settings.EditorRoots)
	settings.PinnedProjects = slices.Clone(settings.PinnedProjects)
	settings.LocalFavorites = maps.Clone(settings.LocalFavorites)
	settings.HiddenProjects = maps.Clone(settings.HiddenProjects)
	win := a.win
	go func() {
		started := time.Now()
		editors, projects, hubs := scanWithVCS(settings, false)
		externalEditors, externalProjects := scanExternalEngineAdapters(ctx, engineAdapters())
		editors = append(editors, externalEditors...)
		projects = append(projects, externalProjects...)
		logger.Printf("scan: %d editors, %d projects, %d hubs in %s",
			len(editors), len(projects), len(hubs), time.Since(started).Round(time.Millisecond))
		if ctx.Err() != nil {
			return
		}
		win.Update(func() {
			if generation != a.scanGeneration {
				return
			}
			a.editors, a.projects, a.hubs = editors, projects, hubs
			a.clearConfirmedDownloadJobs()
			a.scanning = false
			if a.scanAgain {
				a.scanAgain = false
				a.refresh()
				return
			}
			a.enrichProjectVCS(ctx, generation, projects)
		})
	}()
}

func (a *app) enrichProjectVCS(ctx context.Context, generation uint64, projects []projectEntry) {
	win := a.win
	paths := make([]string, len(projects))
	for i := range projects {
		paths[i] = projects[i].Path
	}
	jobs := make(chan string)
	for i := 0; i < 4; i++ {
		go func() {
			for path := range jobs {
				if ctx.Err() != nil {
					return
				}
				info := detectVSCCached(path)
				if ctx.Err() != nil {
					return
				}
				win.Update(func() {
					if generation != a.scanGeneration {
						return
					}
					for i := range a.projects {
						if strings.EqualFold(a.projects[i].Path, path) {
							a.projects[i].VCS = info
							break
						}
					}
				})
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, path := range paths {
			select {
			case <-ctx.Done():
				return
			case jobs <- path:
			}
		}
	}()
}

// hubChanged runs when a hub writes the files this app shares with it.
func (a *app) hubChanged() {
	if time.Since(a.hubWriteAt) < 1500*time.Millisecond {
		return
	}
	a.refresh()
}

func (a *app) save() {
	if err := savePrefs(a.prefs); err != nil {
		a.alert(err.Error())
	}
}

func (a *app) view(c *ui.Context) {
	c.SetTheme(starterTheme)
	if !a.modalOpen() {
		for i, adapter := range engineAdapters() {
			key, ok := engineShortcutKey(i)
			if !ok {
				break
			}
			if c.Shortcut(ui.Cmd, key) {
				a.switchEngine(adapter.ID)
				break
			}
		}
		if c.Shortcut(ui.Cmd, ui.KeyComma) {
			a.page = pageSettings
		}
	}
	t := c.Theme()
	bar := c.TitleBar()
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(36).Shrink(0).Padding(0, bar.Right+12, 0, bar.Left+14).Gap(8).AlignItems(ui.Center).Background(t.Background).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).DragWindow().Children(func() {
			if s := a.engineStrategy(); s.BrandIcon != nil {
				ui.Image(c, s.BrandIcon).Size(14, 14)
			} else {
				ui.Icon(c, s.Icon).Size(14, 14).TextColor(t.TextMuted)
			}
			ui.Text(c, locales.AppTitle()).FontSize(12).TextColor(t.TextMuted).SingleLine()
			ui.Box(c).Height(18).Padding(0, 6).Radius(999).Background(t.SurfaceHover).Border(1, t.Border).Center().Children(func() {
				ui.Text(c, "v"+appVersion).FontSize(9).TextColor(t.TextMuted).SingleLine()
			})
		})
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.engineRail(c)
			ui.Column(c).Fill().Width(220).MinHeight(0).Background(t.Surface).Children(func() {
				a.navigationHeader(c)
				s := a.engineStrategy()
				downloads := s.Pages.Downloads
				ui.Scroll(c).Grow(1).Shrink(1).MinHeight(0).Children(func() {
					ui.Column(c).Padding(8, 10, 16, 10).Gap(2).FocusGroup(ui.Vertical).Children(func() {
						engineCount := func() int {
							count := 0
							for _, p := range a.projects {
								if p.Kind == a.activeEngine && !a.projectHidden(p.Path) {
									count++
								}
							}
							return count
						}
						installCount := func() int {
							count := 0
							for _, e := range a.editors {
								if e.Kind == a.activeEngine {
									count++
								}
							}
							return count
						}
						badge := func(n int) func() {
							if n <= 0 {
								return nil
							}
							return func() { ui.Badge(c, fmt.Sprint(n)) }
						}
						// Each adapter owns its navigation. Unity has no game
						// publishing; Tuanjie does. An imported adapter can add
						// links pages without the view learning about them.
						for _, item := range s.Nav {
							item := item
							var trailing func()
							switch item.Kind {
							case "projects":
								trailing = badge(engineCount())
							case "installs":
								trailing = badge(installCount())
							case "activity":
								trailing = func() {
									if n := a.workspaceTaskCount(); n > 0 {
										ui.Badge(c, fmt.Sprint(n))
									}
								}
							}
							a.navItem(c, item.Route, item.Icon, item.TitleText(), trailing)
						}
					})
				})
				if a.hasLicenseStatus(s) {
					a.licenseStatusStrip(c)
				}
				ui.Divider(c)
				ui.Column(c).Shrink(0).Padding(8, 10, 10, 10).FocusGroup(ui.Vertical).Children(func() {
					if a.hasDownloadsFooter(s) {
						a.navItem(c, downloads, iconDownload, locales.SidebarDownloads(), nil)
					}
					a.navItem(c, pageSettings, iconGear, locales.SidebarSettings(), nil)
				})
			})
			ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Fill().Children(func() {
				a.unityLicenseBanner(c)
				a.uninstallGuardBanner(c)
				a.workspaceActivityBanner(c)
				a.updateBanner(c)
				if adapter, page, ok := enginePageForRoute(a.page); ok && !adapter.BuiltIn {
					a.externalPage(c, adapter, page)
					return
				}
				switch a.page {
				case pageProjects, pageFavorites, pageTuanjieProjects, pageTuanjieFavorites:
					a.projectsPage(c)
				case pageInstalls, pageTuanjieInstalls:
					a.installsPage(c)
				case pageDownloads, pageTuanjieDownloads:
					a.downloadsPage(c)
				case pageStore:
					a.storePage(c)
				case pagePublish:
					a.publishPage(c)
				case pageLicense:
					a.licensePage(c)
				case pageDocs:
					a.docsPage(c)
				case pageLearn:
					a.learnPage(c)
				case pageFeedback:
					a.feedbackPage(c)
				case pageSettings:
					a.settingsPage(c)
				case pageEngineSettings:
					a.engineSettingsPage(c)
				case pageActivity:
					a.activityPage(c)
				default:
					a.projectsPage(c)
				}
			})
		})
	})
	a.newProjectDialog(c)
	a.projectVersionDialog(c)
	a.installedModulesDialog(c)
	a.editorManageDialog(c)
	a.uninstallGuardDialog(c)
	a.availableUpdateDialog(c)
	a.unityLicenseDialog(c)
	a.alertDialog(c)
}

// engineRail is the vertical engine switcher at the far left, as Tuanjie
// Hub's: every registered adapter stacks from the top, the active one wears a
// raised tile, and the rest of the rail is empty. It never scrolls with the
// nav.
func (a *app) engineRail(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(64).Shrink(0).Padding(12, 0, 0, 0).Background(t.Background).
		BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Scroll(c).Grow(1).Shrink(1).MinHeight(0).Children(func() {
			ui.Column(c).Gap(6).FocusGroup(ui.Vertical).Children(func() {
				for i, adapter := range engineAdapters() {
					a.engineRailTab(c, adapter, i)
				}
			})
		})
	})
}

// engineRailTab is one engine's square tab in the rail. The label keeps the
// control named for tests and assistive technology even though the rail
// shows only the brand mark.
func (a *app) engineRailTab(c *ui.Context, adapter *engineAdapter, index int) {
	t := c.Theme()
	s := adapter.strategy()
	engine := adapter.ID
	active := a.activeEngine == engine
	hint := s.Name
	if index < 9 {
		hint += fmt.Sprintf(" · Ctrl+%d", index+1)
	}
	b := ui.Button(c, "").Size(64, 56).Padding(0).
		Label(adapter.DisplayName()).Tooltip(hint).
		Background(ui.Color{}).Border(0, ui.Color{}).
		OnClick(func() { a.switchEngine(engine) })
	var tile, mark ui.Element
	b.Children(func() {
		bg := ui.Color{}
		if active {
			bg = t.SurfaceHover
		}
		tile = ui.Box(c).Size(44, 40).Radius(9).Background(bg).Center().Children(func() {
			// A brand that ships its own faceted art keeps it in both states,
			// muted when another engine is active; monochrome marks follow the
			// text color.
			if s.BrandIcon != nil {
				mark = ui.Image(c, s.BrandIcon).Size(26, 26)
				return
			}
			color := t.TextMuted
			if active || b.Focused() {
				color = t.Text
			}
			mark = ui.Icon(c, s.Icon).Size(25, 25).TextColor(color)
		})
		if active {
			tile.Border(1, t.Border)
		}
	})
	// Hover is read after the children are built, as a button's face does.
	if b.Hovered() {
		if active {
			tile.Background(t.SurfacePressed)
		} else {
			tile.Background(t.Surface)
		}
	}
	if s.BrandIcon != nil {
		opacity := float32(1)
		if !active {
			opacity = 0.72
			if b.Hovered() || b.Focused() {
				opacity = 0.95
			}
		}
		mark.Opacity(opacity)
	}
}

// navigationHeader mirrors the account row at the top of Tuanjie Hub's nav
// column: brand mark, the current engine, and settings at the right. The
// mark names the engine the rail chose, so the column never loses context.
func (a *app) navigationHeader(c *ui.Context) {
	t := c.Theme()
	s := a.engineStrategy()
	ui.Row(c).Height(60).Shrink(0).Padding(14, 12, 12, 14).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(30, 30).Radius(15).Background(t.SurfaceHover).Border(1, t.Border).Center().Children(func() {
			if s.BrandIcon != nil {
				ui.Image(c, s.BrandIcon).Size(18, 18)
				return
			}
			ui.Icon(c, s.Icon).Size(17, 17).TextColor(t.Text)
		})
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, channelName(a.activeEngine)).FontSize(13).FontWeight(600).SingleLine().Clip().Tooltip(s.Name)
			ui.Text(c, locales.AppTitle()).FontSize(10).TextColor(t.TextMuted).SingleLine().Clip()
		})
		gear := ui.Button(c, "").Size(28, 28).Padding(0).Radius(6).
			Label(locales.EngineSettings()).Tooltip(locales.EngineSettings()).
			Background(ui.Color{}).Border(0, ui.Color{}).
			OnClick(func() { a.page = pageEngineSettings })
		gear.Children(func() {
			color := t.TextMuted
			if a.page == pageEngineSettings {
				color = t.Text
			}
			ui.Icon(c, iconGear).Size(17, 17).TextColor(color)
		})
		if a.page == pageEngineSettings {
			gear.Background(t.SurfaceHover)
		}
		if gear.Hovered() {
			gear.Background(t.SurfaceHover)
		}
	})
	ui.Divider(c)
}

// modalOpen reports whether a dialog owns the window, so global navigation
// shortcuts leave its context alone.
func (a *app) modalOpen() bool {
	return a.alertOpen || a.availableOpen || a.manageOpen || a.installedModulesOpen ||
		a.versionOpen || a.newOpen || a.uninstallGuardOpen || a.licenseFlowOpen
}

func (a *app) switchEngine(engine string) {
	if !engineValid(engine) {
		return
	}
	if engine == a.activeEngine {
		return
	}
	old := engineAdapterByID(a.activeEngine)
	next := engineAdapterByID(engine)
	if next == nil {
		return
	}
	kind := ""
	if old != nil {
		if p, ok := old.PageForRoute(a.page); ok {
			kind = p.Kind
		}
	}
	switch {
	case kind != "":
		if p, ok := next.PageForKind(kind); ok {
			a.page = p.Route
		} else {
			a.page = next.LandingPage()
		}
	case a.page == pageSettings || a.page == pageEngineSettings:
		// System and engine settings belong to the launcher context and stay
		// open while the active engine changes.
	default:
		if _, ok := next.PageForRoute(a.page); !ok {
			a.page = next.LandingPage()
		}
	}
	a.activeEngine = engine
	a.prefs.ActiveEngine = engine
	if a.win != nil {
		a.save()
	}
}

// engineForPage names the engine whose page is shown. Routes unique to one
// adapter identify it even when tests set the page alone; shared service pages
// follow the active engine.
func (a *app) engineForPage(page string) string {
	var owner *engineAdapter
	for _, adapter := range engineAdapters() {
		if _, ok := adapter.PageForRoute(page); ok {
			if owner != nil {
				return a.activeEngine
			}
			owner = adapter
		}
	}
	if owner != nil {
		return owner.ID
	}
	return a.activeEngine
}

func (a *app) favoritesPage() bool {
	if _, p, ok := enginePageForRoute(a.page); ok {
		return p.Kind == "favorites"
	}
	return a.page == pageFavorites || a.page == pageTuanjieFavorites
}

// hasDownloadsFooter keeps the pinned downloads entry for engines that
// actually have a downloads workspace; a links-only adapter does not.
func (a *app) hasDownloadsFooter(s engineStrategy) bool {
	if s.Adapter == nil {
		return true
	}
	if !s.Adapter.Features.Downloads {
		return false
	}
	_, ok := s.Adapter.PageForKind("downloads")
	return ok
}

// hasLicenseStatus keeps license text on engines that actually use a license
// workspace. Godot and links-only adapters should not show "free license".
func (a *app) hasLicenseStatus(s engineStrategy) bool {
	if s.Adapter == nil {
		return true
	}
	return s.Adapter.ID == kindUnity || s.Adapter.ID == kindTuanjie
}

// header is the title of a page, with its count and its actions.
func (a *app) header(c *ui.Context, title, subtitle string, actions func()) {
	t := c.Theme()
	ui.Column(c).Padding(24, 24, 18, 24).Gap(5).Shrink(0).Children(func() {
		ui.Text(c, strings.ToUpper(channelName(a.activeEngine))).FontSize(10).FontWeight(600).
			LetterSpacing(1.2).TextColor(t.TextMuted).SingleLine().Clip().Tooltip(channelName(a.activeEngine))
		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).Grow(1).MinWidth(0).FontSize(24).FontWeight(600).SingleLine().Clip().Tooltip(title)
			if actions != nil {
				ui.Row(c).Gap(8).Shrink(0).Children(actions)
			}
		})
		if subtitle != "" {
			ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted).SingleLine().Clip().Tooltip(subtitle)
		}
	})
}

func (a *app) emptyState(c *ui.Context, title, hint string) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(8).Padding(40).Children(func() {
		ui.Box(c).Size(56, 56).Radius(12).Background(t.SurfaceHover).Border(1, t.Border).Center().Margin(0, 0, 8, 0).Children(func() {
			ui.Icon(c, iconForPage(a.page)).Size(28, 28).TextColor(t.TextMuted)
		})
		ui.Text(c, title).FontSize(16).Bold().TextColor(t.TextMuted)
		ui.Text(c, hint).FontSize(12).TextColor(t.TextMuted).TextAlign(ui.Center).MaxWidth(420)
	})
}

func iconForPage(page string) *ui.SVG {
	switch page {
	case pageInstalls, pageTuanjieInstalls:
		return iconInstalls
	case pageDownloads, pageTuanjieDownloads, pageActivity:
		return iconDownload
	case pageFavorites, pageTuanjieFavorites:
		return iconFavorite
	default:
		return iconProjects
	}
}

func (a *app) formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	if !mygo.App.IsReady() {
		return t.Format("2006-01-02")
	}
	formats := mygo.App.Region().FormatLocale(mygo.App.Locale())
	return i18n.FormatDate(formats, t, i18n.DateOptions{Skeleton: "yMMMd"})
}

// alert reports a problem in the app's own UI and is safe from workers.
func (a *app) alert(message string) {
	logger.Printf("alert: %s", message)
	win := a.win
	if win == nil {
		a.alertMessage, a.alertOpen = message, true
		return
	}
	win.Update(func() { a.alertMessage, a.alertOpen = message, true })
}

func (a *app) alertDialog(c *ui.Context) {
	if !a.alertOpen {
		return
	}
	_, height := c.Size()
	modal(c, &a.alertOpen, func() {
		ui.Column(c).Width(460).Gap(16).Children(func() {
			ui.Text(c, locales.AlertTitle()).FontSize(18).Bold()
			ui.Scroll(c).MaxHeight(max(100, height-220)).Children(func() { ui.Text(c, a.alertMessage).FontSize(13).MaxWidth(460) })
			ui.Row(c).Justify(ui.End).Children(func() {
				ui.PrimaryButton(c, locales.DialogOK()).OnClick(func() { a.alertOpen = false })
			})
		})
	})
}

// pickFolder asks for a folder, and runs fn with it on the main thread.
func (a *app) pickFolder(title string, fn func(string)) {
	win := a.win
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: title, Directory: true})
		if err != nil || len(paths) == 0 {
			return
		}
		dir := paths[0]
		win.Update(func() { fn(dir) })
	}()
}

func (a *app) openProject(p projectEntry) {
	a.openProjectWithEditor(p, editorInstall{Exe: p.Editor, Version: p.Version, Kind: p.Kind})
}

// expected is the editor identity presented to the user when the action was
// chosen. Rechecking the current scan prevents a replaced editor at the same
// path from silently turning an ordinary open into a version migration.
func (a *app) openProjectWithEditor(p projectEntry, expected editorInstall) {
	a.openProjectWithEditorChecked(p, expected, false)
}

func (a *app) openProjectWithEditorChecked(p projectEntry, expected editorInstall, licensed bool) {
	if a.uninstallBusy && strings.EqualFold(p.Editor, a.manageEditor.Exe) {
		a.alert(locales.InstallUninstallBusy())
		return
	}
	if p.Editor == "" {
		a.alert(locales.EditorMissing(p.Version))
		return
	}
	if p.Kind == kindGodot {
		a.openGodotProject(p, expected)
		return
	}
	if isExternalEngine(p.Kind) {
		a.openExternalProject(p, expected)
		return
	}
	version, _, ok := readProject(p.Path)
	if !ok {
		a.alert(locales.NotAProject(p.Path))
		a.refresh()
		return
	}
	if !strings.EqualFold(version, p.Version) {
		a.alert(locales.ProjectVersionChanged())
		a.refresh()
		return
	}
	current, ok := a.editorInstall(p.Editor)
	if !ok || !strings.EqualFold(expected.Exe, p.Editor) || !strings.EqualFold(current.Version, expected.Version) || current.Kind != expected.Kind || current.Kind != p.Kind {
		a.alert(locales.ProjectEditorChanged())
		a.refresh()
		return
	}
	if current.Simulated {
		a.alert(locales.SimulatedOpen())
		return
	}
	if !licensed && current.Kind == kindUnity {
		if actual := editorExecutableVersion(p.Editor); actual != "" && !strings.EqualFold(actual, expected.Version) {
			a.alert(locales.ProjectEditorChanged())
			a.refresh()
			return
		}
		a.withUnityLicense(current, func() { a.openProjectWithEditorChecked(p, expected, true) })
		return
	}
	if err := launchVersionCheckedEditor(p.Editor, expected.Version, []string{"-projectPath", p.Path}, filepath.Dir(p.Editor), a.launchEditor); err != nil {
		if errors.Is(err, errEditorVersionChanged) {
			a.alert(locales.ProjectEditorChanged())
			a.refresh()
			return
		}
		a.alert(locales.OpenFailed(err.Error()))
		return
	}
	if !strings.EqualFold(expected.Version, p.Version) {
		a.watchProjectVersion(p.Path, p.Version)
	}
}

// openGodotProject uses Godot's own --path convention; Godot projects have
// project.godot rather than Unity's ProjectVersion.txt.
func (a *app) openGodotProject(p projectEntry, expected editorInstall) {
	current, ok := a.editorInstall(p.Editor)
	if !ok || !strings.EqualFold(current.Exe, expected.Exe) || current.Kind != kindGodot {
		a.alert(locales.ProjectEditorChanged())
		a.refresh()
		return
	}
	if err := startDetached(current.Exe, []string{"--path", p.Path}, filepath.Dir(current.Exe)); err != nil {
		a.alert(locales.OpenFailed(err.Error()))
	}
}

// openExternalProject hands a project to its engine adapter. An adapter may
// use any CLI convention; the launcher never guesses Unity flags for it.
func (a *app) openExternalProject(p projectEntry, expected editorInstall) {
	adapter := engineAdapterByID(p.Kind)
	if adapter == nil {
		a.alert(locales.ProjectEditorChanged())
		return
	}
	current, ok := a.editorInstall(p.Editor)
	if !ok || !strings.EqualFold(current.Exe, expected.Exe) || !strings.EqualFold(current.Version, expected.Version) || current.Kind != adapter.ID {
		a.alert(locales.ProjectEditorChanged())
		a.refresh()
		return
	}
	if adapter.Process == nil {
		a.alert(locales.EnginesNoLaunch())
		return
	}
	run := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return launchViaEngineAdapter(ctx, adapter, current.Exe, p.Path, nil)
	}
	if a.win == nil {
		if err := run(); err != nil {
			a.alert(locales.OpenFailed(err.Error()))
		}
		return
	}
	go func() {
		if err := run(); err != nil {
			a.win.Update(func() { a.alert(locales.OpenFailed(err.Error())) })
		}
	}()
}

func (a *app) addProject() {
	a.addProjectForEngine("")
}

func (a *app) addProjectForEngine(engine string) {
	a.pickFolder(locales.DialogAddProject(), func(dir string) {
		if !isProject(dir) {
			a.alert(locales.NotAProject(dir))
			return
		}
		if version, _, ok := readProject(dir); engine != "" && ok && kindOfVersion(version) != engine {
			a.alert(locales.ProjectWrongEngine())
			return
		}
		a.prefs.PinnedProjects = dedupe(append(a.prefs.PinnedProjects, dir))
		forgetProjectAliases(a.prefs.HiddenProjects, dir)
		a.save()
		a.pushToHub(dir)
		a.refresh()
	})
}

// hubFor is the hub a project's star and additions go to: the one that
// lists it, else the Unity Hub, which holds a project list this app writes.
func (a *app) hubFor(p projectEntry) (hub, bool) {
	if a.prefs.NoHubSync {
		return hub{}, false
	}
	for _, h := range a.hubs {
		if h.Kind == p.Kind {
			return h, true
		}
	}
	return hub{}, false
}

// Favorites work locally even when the matching Hub is unavailable or sync is disabled.
func (a *app) toggleFavorite(p projectEntry) {
	h, ok := a.hubFor(p)
	on := !p.Favorite
	key := localProjectKey(p.Path)
	if ok {
		if h.Kind != p.Hub && h.writable() {
			if err := h.addProject(p); err != nil {
				a.alert(locales.HubWriteFailed(err.Error()))
				return
			}
		}
		if err := h.setFavorite(p.Path, on); err != nil {
			a.alert(locales.HubWriteFailed(err.Error()))
			return
		}
		a.hubWriteAt = time.Now()
		if len(a.prefs.LocalFavorites) > 0 {
			forgetProjectAliases(a.prefs.LocalFavorites, p.Path)
			a.save()
		}
	} else {
		if a.prefs.LocalFavorites == nil {
			a.prefs.LocalFavorites = map[string]bool{}
		}
		previous := make(map[string]bool, len(a.prefs.LocalFavorites))
		for path, favorite := range a.prefs.LocalFavorites {
			previous[path] = favorite
		}
		forgetProjectAliases(a.prefs.LocalFavorites, p.Path)
		a.prefs.LocalFavorites[key] = on
		if err := savePrefs(a.prefs); err != nil {
			a.prefs.LocalFavorites = previous
			a.alert(err.Error())
			return
		}
	}
	for i := range a.projects {
		if a.projects[i].Path != p.Path {
			continue
		}
		a.projects[i].Favorite = on
		if ok {
			a.projects[i].Hub = h.Kind
		}
	}
}

// pushToHub adds a project this app added to the hub that owns its engine,
// so that the hub lists it too.
func (a *app) pushToHub(dir string) {
	if a.prefs.NoHubSync {
		return
	}
	version, info, ok := readProject(dir)
	if !ok {
		return
	}
	entry := projectEntry{Path: filepath.Clean(dir), Name: filepath.Base(dir), Version: version, Kind: kindOfVersion(version)}
	if info != nil {
		entry.Modified = info.ModTime()
	}
	added, err := exportProjects(a.hubs, []projectEntry{entry})
	if err != nil {
		a.alert(locales.HubWriteFailed(err.Error()))
		return
	}
	if added > 0 {
		a.hubWriteAt = time.Now()
	}
}

// editorInstall is the installed editor of an executable, whose version a
// project created with it has.
func (a *app) editorInstall(exe string) (editorInstall, bool) {
	for _, e := range a.editors {
		if strings.EqualFold(e.Exe, exe) {
			return e, true
		}
	}
	return editorInstall{}, false
}

// editorIsSimulated reports whether an executable is a stand-in.
func (a *app) editorIsSimulated(exe string) bool {
	e, ok := a.editorInstall(exe)
	return ok && e.Simulated
}

// startHubWatch follows the hubs' files while the app runs, so that a change
// made in a hub shows here without a manual refresh.
func (a *app) startHubWatch() {
	hubs := detectHubs()
	ctx, cancel := context.WithCancel(context.Background())
	a.hubCancel = cancel
	win := a.win
	go func() {
		previous := recentEditorProjects()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current := recentEditorProjects()
				if !slices.Equal(current, previous) {
					previous = current
					win.Update(a.refresh)
				}
				pending := pendingProjects()
				guards := uninstallGuards()
				var ready []pendingProject
				for _, p := range pending {
					if pendingProjectReady(p) {
						ready = append(ready, p)
					}
				}
				win.Update(func() { a.pendingProjectList = pending; a.uninstallGuards = guards; a.finishPendingProjects(ready) })
			}
		}
	}()
	watchHubs(ctx, hubs, func() {
		win.Update(a.hubChanged)
	})
}
