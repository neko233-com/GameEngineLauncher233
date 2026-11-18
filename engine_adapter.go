package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"unitystarter/locales"
)

// engineAdapterSchemaVersion is the manifest format this build understands.
// A manifest from the future is refused rather than half-interpreted.
const engineAdapterSchemaVersion = 1

// engineAdapterFeatures are the optional behaviours a page or an action may
// rely on. Built-ins enable what they already do; external adapters declare
// what their process can do.
type engineAdapterFeatures struct {
	LicenseFlow   bool `json:"licenseFlow,omitempty"`
	CreateProject bool `json:"createProject,omitempty"`
	Downloads     bool `json:"downloads,omitempty"`
	Templates     bool `json:"templates,omitempty"`
}

// engineAdapterURLs are engine-owned destinations used by the shared link
// pages, so an adapter never has to imitate another engine's URLs.
type engineAdapterURLs struct {
	AssetStore string `json:"assetStore,omitempty"`
	Docs       string `json:"docs,omitempty"`
	License    string `json:"license,omitempty"`
	Learn      string `json:"learn,omitempty"`
	Community  string `json:"community,omitempty"`
}

// engineAdapterProcess declares an optional JSON-RPC process, in the spirit
// of a language server: the launcher starts it, initializes it and asks it to
// detect editors/projects or launch a project.
type engineAdapterProcess struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

// engineLink is one destination of a manifest-defined page.
type engineLink struct {
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	URL    string `json:"url"`
}

// engineLinkGroup is a titled group of links inside a manifest page.
type engineLinkGroup struct {
	Title string       `json:"title"`
	Links []engineLink `json:"links"`
}

// enginePage is a navigation destination owned by one adapter. Built-ins use
// stable route constants; user adapters get /engine/<id>/<page-id> routes.
type enginePage struct {
	Key    string
	Route  string
	Kind   string
	Icon   *ui.SVG
	Groups []engineLinkGroup
	URL    string
	title  func() string
}

// EnginePageKinds lists the content handlers a manifest page may choose.
var enginePageKinds = map[string]bool{
	"projects": true, "installs": true, "downloads": true, "favorites": true,
	"activity": true, "links": true, "publish": true, "license": true,
	"learn": true, "docs": true, "feedback": true, "store": true,
}

// engineAdapter is one engine the launcher knows. Built-ins carry localized
// titles and the existing Go handlers; user adapters carry literal manifest
// strings and link pages, plus an optional detection/launch process.
type engineAdapter struct {
	ID          string
	Version     string
	Description string
	Homepage    string
	Source      string
	Icon        *ui.SVG
	BrandIcon   *ui.SVG
	Pages       []enginePage
	Features    engineAdapterFeatures
	URLs        engineAdapterURLs
	Process     *engineAdapterProcess
	BuiltIn     bool
	Dir         string

	name func() string
}

func (a *engineAdapter) DisplayName() string {
	if a == nil {
		return ""
	}
	if a.name != nil {
		return a.name()
	}
	return a.ID
}

func (a *engineAdapter) PageForRoute(route string) (enginePage, bool) {
	for _, p := range a.Pages {
		if p.Route == route {
			return p, true
		}
	}
	return enginePage{}, false
}

func (a *engineAdapter) PageForKind(kind string) (enginePage, bool) {
	for _, p := range a.Pages {
		if p.Kind == kind {
			return p, true
		}
	}
	return enginePage{}, false
}

func (a *engineAdapter) LandingPage() string {
	if p, ok := a.PageForKind("projects"); ok {
		return p.Route
	}
	for _, p := range a.Pages {
		if p.Kind != "downloads" {
			return p.Route
		}
	}
	if len(a.Pages) > 0 {
		return a.Pages[0].Route
	}
	return pageProjects
}

func (a *engineAdapter) NavPages() []enginePage {
	out := make([]enginePage, 0, len(a.Pages))
	for _, p := range a.Pages {
		if p.Kind != "downloads" {
			out = append(out, p)
		}
	}
	return out
}

func (p enginePage) TitleText() string {
	if p.title != nil {
		return p.title()
	}
	return p.Key
}

func (a *engineAdapter) strategy() engineStrategy {
	routes := engineRoutePages{}
	for _, p := range a.Pages {
		switch p.Kind {
		case "projects":
			routes.Projects = p.Route
		case "installs":
			routes.Installs = p.Route
		case "downloads":
			routes.Downloads = p.Route
		case "favorites":
			routes.Favorites = p.Route
		}
	}
	if routes.Projects == "" {
		routes.Projects = a.LandingPage()
	}
	if routes.Installs == "" {
		routes.Installs = routes.Projects
	}
	if routes.Favorites == "" {
		routes.Favorites = routes.Projects
	}
	if routes.Downloads == "" {
		routes.Downloads = routes.Projects
	}
	return engineStrategy{
		engineBranding: engineBranding{
			Name:      a.DisplayName(),
			Icon:      a.Icon,
			BrandIcon: a.BrandIcon,
		},
		Pages:            routes,
		Nav:              a.NavPages(),
		Adapter:          a,
		NeedsLicenseFlow: a.Features.LicenseFlow,
		AssetStoreURL:    a.URLs.AssetStore,
		DocsURL:          a.URLs.Docs,
		LicenseURL:       a.URLs.License,
		LearnURL:         a.URLs.Learn,
		CommunityURL:     a.URLs.Community,
	}
}

// engineRegistryOrder and engineRegistryByID hold the built-in adapters plus
// the user's installed manifests. The launcher is single-window; the mutex
// keeps the scan worker from reading a half-reloaded registry.
var (
	engineRegistryMu     sync.RWMutex
	engineRegistryOrder  []string
	engineRegistryByID   map[string]*engineAdapter
	engineRegistryLoaded time.Time
)

func init() {
	resetEngineRegistry()
}

func resetEngineRegistry() {
	engineRegistryMu.Lock()
	defer engineRegistryMu.Unlock()
	engineRegistryByID = map[string]*engineAdapter{}
	engineRegistryOrder = nil
	for _, a := range []*engineAdapter{builtinUnityAdapter(), builtinTuanjieAdapter(), builtinGodotAdapter()} {
		engineRegistryByID[a.ID] = a
		engineRegistryOrder = append(engineRegistryOrder, a.ID)
	}
	engineRegistryLoaded = time.Now()
}

func addEngineAdapterLocked(a *engineAdapter) {
	if a == nil || a.ID == "" {
		return
	}
	if _, exists := engineRegistryByID[a.ID]; exists {
		return
	}
	engineRegistryByID[a.ID] = a
	engineRegistryOrder = append(engineRegistryOrder, a.ID)
}

func engineAdapters() []*engineAdapter {
	engineRegistryMu.RLock()
	defer engineRegistryMu.RUnlock()
	out := make([]*engineAdapter, 0, len(engineRegistryOrder))
	for _, id := range engineRegistryOrder {
		if a := engineRegistryByID[id]; a != nil {
			out = append(out, a)
		}
	}
	return out
}

func engineAdapterByID(id string) *engineAdapter {
	engineRegistryMu.RLock()
	defer engineRegistryMu.RUnlock()
	return engineRegistryByID[id]
}

func engineAdapterRoutePage(route string) (*engineAdapter, enginePage, bool) {
	for _, a := range engineAdapters() {
		if p, ok := a.PageForRoute(route); ok {
			return a, p, true
		}
	}
	return nil, enginePage{}, false
}

// pageDef is the terse declaration of a built-in adapter page.
func pageDef(route, kind string, icon *ui.SVG, title func() string) enginePage {
	return enginePage{Key: strings.TrimPrefix(route, "/"), Route: route, Kind: kind, Icon: icon, title: title}
}

func builtinUnityAdapter() *engineAdapter {
	return &engineAdapter{
		ID:          kindUnity,
		Version:     appVersion,
		Description: locales.EnginesUnityDescription(),
		Homepage:    urlRepo,
		BuiltIn:     true,
		Icon:        iconUnity,
		name:        locales.ChannelUnity,
		Features:    engineAdapterFeatures{LicenseFlow: true, CreateProject: true, Downloads: true, Templates: true},
		URLs: engineAdapterURLs{
			AssetStore: urlAssetStoreUnity,
			Docs:       urlDocsUnity,
			License:    urlLicenseUnity,
			Learn:      "https://learn.unity.com/",
			Community:  urlCommunity,
		},
		Pages: []enginePage{
			pageDef(pageProjects, "projects", iconProjects, locales.SidebarProjects),
			pageDef(pageInstalls, "installs", iconInstalls, locales.SidebarInstalls),
			pageDef(pageFavorites, "favorites", iconFavorite, locales.SidebarFavorites),
			pageDef(pageLearn, "learn", iconLearn, locales.PageLearn),
			pageDef(pageStore, "store", iconStore, locales.PageStore),
			pageDef(pageLicense, "license", iconLicense, locales.PageLicense),
			pageDef(pageActivity, "activity", iconDownload, locales.ActivityTitle),
			pageDef(pageDocs, "docs", iconDocs, locales.PageDocs),
			pageDef(pageFeedback, "feedback", iconFeedback, locales.PageFeedback),
			pageDef(pageDownloads, "downloads", iconDownload, locales.SidebarDownloads),
		},
	}
}

func builtinGodotAdapter() *engineAdapter {
	return &engineAdapter{
		ID:          kindGodot,
		Version:     appVersion,
		Description: locales.EnginesGodotDescription(),
		Homepage:    "https://godotengine.org/",
		BuiltIn:     true,
		Icon:        iconGodot,
		BrandIcon:   iconGodotBrand,
		name:        func() string { return "Godot" },
		Features:    engineAdapterFeatures{LicenseFlow: false, CreateProject: false, Downloads: false, Templates: false},
		URLs: engineAdapterURLs{
			Docs:      "https://docs.godotengine.org/en/stable/",
			Community: "https://forum.godotengine.org/",
			Learn:     "https://docs.godotengine.org/en/stable/community/tutorials.html",
		},
		Pages: []enginePage{
			pageDef(pageGodotProjects, "projects", iconProjects, locales.SidebarProjects),
			pageDef(pageGodotInstalls, "installs", iconInstalls, locales.SidebarInstalls),
			pageDef(pageGodotFavorites, "favorites", iconFavorite, locales.SidebarFavorites),
			{
				Key: "get", Route: pageGodotGet, Kind: "links", Icon: iconDownload,
				title: locales.GodotGet,
				Groups: []engineLinkGroup{{
					Title: "Godot Engine",
					Links: []engineLink{
						{Title: locales.GodotDownload(), Detail: locales.GodotDownloadHint(), URL: "https://godotengine.org/download/windows/"},
						{Title: locales.GodotAssetLibrary(), Detail: locales.GodotAssetLibraryHint(), URL: "https://godotengine.org/asset-library/asset"},
					},
				}},
			},
			{
				Key: "docs", Route: pageGodotDocs, Kind: "links", Icon: iconDocs,
				title: locales.PageDocs,
				Groups: []engineLinkGroup{{
					Title: locales.PageDocs(),
					Links: []engineLink{
						{Title: locales.GodotManual(), Detail: locales.GodotManualHint(), URL: "https://docs.godotengine.org/en/stable/"},
						{Title: locales.GodotScripting(), Detail: locales.GodotScriptingHint(), URL: "https://docs.godotengine.org/en/stable/tutorials/scripting/gdscript/"},
					},
				}},
			},
			{
				Key: "community", Route: pageGodotCommunity, Kind: "links", Icon: iconFeedback,
				title: locales.LinkCommunity,
				Groups: []engineLinkGroup{{
					Title: locales.LinkCommunity(),
					Links: []engineLink{
						{Title: "Forum", Detail: locales.LinkCommunityHint(), URL: "https://forum.godotengine.org/"},
						{Title: locales.GodotChat(), Detail: locales.GodotChatHint(), URL: "https://godotengine.org/community/"},
					},
				}},
			},
		},
	}
}

func builtinTuanjieAdapter() *engineAdapter {
	return &engineAdapter{
		ID:          kindTuanjie,
		Version:     appVersion,
		Description: locales.EnginesTuanjieDescription(),
		Homepage:    urlRepo,
		BuiltIn:     true,
		Icon:        iconTuanjie,
		BrandIcon:   iconTuanjieBrand,
		name:        locales.ChannelTuanjie,
		Features:    engineAdapterFeatures{LicenseFlow: false, CreateProject: true, Downloads: true, Templates: true},
		URLs: engineAdapterURLs{
			AssetStore: urlAssetStoreTuanjie,
			Docs:       urlDocsTuanjie,
			License:    urlLicenseTuanjie,
			Learn:      urlLearn,
			Community:  urlCommunity,
		},
		Pages: []enginePage{
			pageDef(pageTuanjieProjects, "projects", iconProjects, locales.SidebarProjects),
			pageDef(pageTuanjieInstalls, "installs", iconInstalls, locales.SidebarInstalls),
			pageDef(pageTuanjieFavorites, "favorites", iconFavorite, locales.SidebarFavorites),
			pageDef(pageLearn, "learn", iconLearn, locales.PageLearn),
			pageDef(pageStore, "store", iconStore, locales.PageStore),
			pageDef(pagePublish, "publish", iconPublish, locales.PagePublish),
			pageDef(pageLicense, "license", iconLicense, locales.PageLicense),
			pageDef(pageActivity, "activity", iconDownload, locales.ActivityTitle),
			pageDef(pageDocs, "docs", iconDocs, locales.PageDocs),
			pageDef(pageFeedback, "feedback", iconFeedback, locales.PageFeedback),
			pageDef(pageTuanjieDownloads, "downloads", iconDownload, locales.SidebarDownloads),
		},
	}
}

// adapterIcon returns a built-in icon for manifest icon names, falling back
// to the launcher's generic engine mark.
func adapterIcon(name string) *ui.SVG {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "projects", "project":
		return iconProjects
	case "installs", "install":
		return iconInstalls
	case "download", "downloads":
		return iconDownload
	case "favorite", "favorites", "star":
		return iconFavorite
	case "learn":
		return iconLearn
	case "store":
		return iconStore
	case "publish":
		return iconPublish
	case "license":
		return iconLicense
	case "docs", "documentation":
		return iconDocs
	case "feedback":
		return iconFeedback
	default:
		return iconEngine
	}
}

func externalPageRoute(id, pageID string) string {
	return "/engine/" + id + "/" + pageID
}

func externalPageRouteID(route string) (string, string, bool) {
	const prefix = "/engine/"
	if !strings.HasPrefix(route, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(route, prefix)
	id, page, ok := strings.Cut(rest, "/")
	if !ok || id == "" || page == "" {
		return "", "", false
	}
	return id, page, true
}

// enginePageForRoute returns the page and adapter that own a route. Shared
// built-in pages belong to whichever engine is active; the caller decides
// that fallback.
func enginePageForRoute(route string) (*engineAdapter, enginePage, bool) {
	return engineAdapterRoutePage(route)
}

func isExternalEngine(kind string) bool {
	a := engineAdapterByID(kind)
	return a != nil && !a.BuiltIn
}

func engineDisplayName(kind string) string {
	if a := engineAdapterByID(kind); a != nil {
		return a.DisplayName()
	}
	return kind
}

// engineShortcutKey maps the first nine adapters to Ctrl+1…Ctrl+9.
func engineShortcutKey(index int) (ui.Key, bool) {
	keys := []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}
	if index < 0 || index >= len(keys) {
		return 0, false
	}
	return keys[index], true
}

func validateManifest(a *engineAdapter) error {
	if a == nil {
		return fmt.Errorf("empty adapter")
	}
	if a.ID == "" || a.DisplayName() == "" {
		return fmt.Errorf("adapter id and name are required")
	}
	if len(a.Pages) == 0 {
		return fmt.Errorf("adapter %q has no pages", a.ID)
	}
	return nil
}
