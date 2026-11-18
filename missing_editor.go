package main

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"unitystarter/locales"
)

// askMissingEditor offers to install the editor version a project needs,
// instead of failing with a dead-end alert as the old flow did.
func (a *app) askMissingEditor(p projectEntry) {
	if p.Kind == "" {
		p.Kind = kindOfVersion(p.Version)
	}
	for _, e := range a.editors {
		if e.Kind == p.Kind && e.Version == p.Version && e.Exe != "" {
			p.Editor = e.Exe
			a.openProject(p)
			return
		}
	}
	a.missingProject = p
	a.missingDownload = downloadEntry{}
	a.missingDownloadFound = false
	a.ensureDownloads(false)
	for _, e := range a.downloads {
		if e.Engine == p.Kind && e.Version == p.Version {
			a.missingDownload, a.missingDownloadFound = e, true
			break
		}
	}
	if !a.missingDownloadFound {
		for _, e := range a.unitySearchEntries {
			if e.Engine == p.Kind && e.Version == p.Version {
				a.missingDownload, a.missingDownloadFound = e, true
				break
			}
		}
	}
	a.missingEditorOpen = true
}

func (a *app) hasEditorForEngine(kind string) bool {
	for _, e := range a.editors {
		if e.Kind == kind && e.Exe != "" && !e.Simulated {
			return true
		}
	}
	return false
}

func (a *app) missingEditorDialog(c *ui.Context) {
	if !a.missingEditorOpen {
		return
	}
	p := a.missingProject
	_, w := c.Size()
	title := locales.MissingEditorTitle()
	hint := locales.MissingEditorHint(displayVersion(p.Version, p.DisplayVersion), channelName(p.Kind))
	modal(c, &a.missingEditorOpen, func() {
		ui.Column(c).Width(min(520, max(320, w-80))).Gap(14).Children(func() {
			ui.Text(c, title).FontSize(18).Bold()
			ui.Text(c, hint).FontSize(13).TextColor(c.Theme().TextMuted)
			if a.missingDownloadFound {
				ui.Text(c, humanSize(a.missingDownload.Size)).FontSize(11).TextColor(c.Theme().TextMuted)
			} else {
				ui.Text(c, locales.MissingEditorNoDownload()).FontSize(12).TextColor(c.Theme().TextMuted)
			}
			ui.Row(c).Gap(8).Wrap().Justify(ui.End).Children(func() {
				ui.Button(c, locales.MissingEditorOpenDownloads()).OnClick(a.openDownloadsForMissingEditor)
				if a.hasEditorForEngine(p.Kind) {
					ui.Button(c, locales.MissingEditorChooseVersion()).OnClick(func() {
						a.missingEditorOpen = false
						a.chooseProjectVersion(p)
					})
				}
				ui.Button(c, locales.ActionCancel()).OnClick(func() { a.missingEditorOpen = false })
				if a.missingDownloadFound {
					ui.PrimaryButton(c, locales.MissingEditorDownload()).OnClick(a.downloadMissingEditor)
				}
			})
		})
	})
}

func (a *app) openDownloadsForMissingEditor() {
	p := a.missingProject
	a.missingEditorOpen = false
	if p.Kind == kindGodot {
		a.page = pageGodotGet
		return
	}
	if isExternalEngine(p.Kind) {
		a.page = engineStrategyFor(p.Kind).Pages.Projects
		return
	}
	a.page = pageDownloads
	if p.Kind == kindTuanjie {
		a.downloadQueryTuanjie = p.Version
		a.downloadFilterTuanjie = ""
	} else {
		a.downloadQueryUnity = p.Version
		a.downloadFilterUnity = ""
		if p.Kind == kindUnity {
			a.searchUnityReleases(p.Version)
		}
	}
}

func (a *app) downloadMissingEditor() {
	p := a.missingProject
	e := a.missingDownload
	a.missingEditorOpen = false
	if err := validateEditorInstallEntry(e); err != nil {
		a.alert(locales.OpenFailed(err.Error()))
		return
	}
	a.pendingProjectPath, a.pendingProjectVersion, a.pendingProjectKind = p.Path, p.Version, p.Kind
	a.page = pageDownloads
	a.startDownload(e)
}

// openPendingProject runs after a refresh. It opens the project only once the
// matching editor has appeared, so an interrupted download can be retried.
func (a *app) openPendingProject() {
	if a.pendingProjectPath == "" {
		return
	}
	var project *projectEntry
	for i := range a.projects {
		p := &a.projects[i]
		if strings.EqualFold(p.Path, a.pendingProjectPath) && p.Kind == a.pendingProjectKind && strings.EqualFold(p.Version, a.pendingProjectVersion) {
			project = p
			break
		}
	}
	if project == nil || project.Editor == "" {
		return
	}
	p := *project
	a.pendingProjectPath, a.pendingProjectVersion, a.pendingProjectKind = "", "", ""
	a.openProject(p)
}
