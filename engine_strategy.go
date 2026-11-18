// engine_strategy.go —— 每个 engine 独立的品牌 + 行为策略。
//
// Unity 和 Tuanjie 不是同构的两个 tab：它们各有自己的官方标志、
// 自己的页面集合、自己的下载源、自己的模板来源、自己的授权机制。
// 这些差异由 engineAdapter 描述，engineStrategy 是视图层使用的只读
// 投影；新增引擎应注册 adapter，不要在 view 层加 if kindUnity。

package main

import "github.com/egoist/mygo/ui"

// engineBranding 描述一个 engine 的品牌标识（视觉层）。
type engineBranding struct {
	Name      string  // "Unity" / "团结引擎" / user adapter name
	Icon      *ui.SVG // monochrome official mark, for tinted contexts
	BrandIcon *ui.SVG // official mark in its own colours, for the rail
}

// engineRoutePages 描述一个 engine 的主路由。缺少的页面回退到首个页面，
// 这样外部适配器只定义 links 页面也能工作。
type engineRoutePages struct {
	Projects  string
	Installs  string
	Downloads string
	Favorites string
}

// engineStrategy 描述一个 engine 的视图投影（品牌 + 页面 + 行为）。
type engineStrategy struct {
	engineBranding
	Pages   engineRoutePages
	Nav     []enginePage
	Adapter *engineAdapter
	// 行为策略 —— 每个 engine 各有一套
	NeedsLicenseFlow bool // Unity 需要授权流程，Tuanjie / 外部引擎可关闭
	DownloadChannel  string
	TemplateSource   string
	AssetStoreURL    string
	DocsURL          string
	LicenseURL       string
	LearnURL         string
	CommunityURL     string
}

// engineStrategyFor 返回指定 engine 的策略；未注册的 id 回退到 Unity，
// 避免旧设置或损坏的适配器把窗口留在空白状态。
func engineStrategyFor(kind string) engineStrategy {
	if a := engineAdapterByID(kind); a != nil {
		return a.strategy()
	}
	if a := engineAdapterByID(kindUnity); a != nil {
		return a.strategy()
	}
	return builtinUnityAdapter().strategy()
}

// engineValid 判断 kind 是否是已注册 engine。
func engineValid(kind string) bool {
	return engineAdapterByID(kind) != nil
}

// engineStrategy 返回当前 activeEngine 的策略。
func (a *app) engineStrategy() engineStrategy {
	return engineStrategyFor(a.activeEngine)
}

// enginePages 返回一个 engine 的四个主路由，兼容旧的调用点。
func enginePages(engine string) (string, string, string, string) {
	s := engineStrategyFor(engine)
	return s.Pages.Projects, s.Pages.Installs, s.Pages.Downloads, s.Pages.Favorites
}
