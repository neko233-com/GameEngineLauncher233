# GameEngineLauncher233 — Agent 执行手册

本文档给后续 Agent 使用。所有构建/安装链路已自动化，Agent 只需按步骤执行。

## 项目概要

- **产品**: GameEngineLauncher233 — 面向 Unity、团结引擎、Godot 与社区适配器的游戏引擎启动器
- **技术栈**: Go + github.com/egoist/mygo（原生 UI，非 Web 前端）
- **仓库**: https://github.com/neko233-com/UnityStarter（main 分支）
- **平台**: Windows amd64（per-user 安装）
- **当前版本**: 1.9.0

## 目录结构

```
GameEngineLauncher233/
├── Taskfile.yml            # ⭐ 所有构建/发布的入口
├── mygo.json               # mygo 构建配置（版本号在这里）
├── icons.go                # SVG 品牌图标（官方 Unity/团结标志 + 通用图标）
├── engine_strategy.go      # 视图使用的 engineStrategy 投影
├── engine_adapter.go       # ⭐ 适配器注册表 + 内置 Unity/团结适配器
├── engine_adapter_manifest.go # ⭐ adapter.json 解析与校验
├── engine_adapter_install.go  # GitHub 导入、安装、加载、移除
├── engine_adapter_process.go  # LSP 风格 stdio JSON-RPC 进程
├── engine_adapter_ui.go    # 引擎适配设置页 + 外部适配器链接页
├── ENGINE_ADAPTERS.md      # ⭐ 社区适配器规范
├── examples/engine-adapter # 可复制的声明式适配器示例
├── app.go                  # 主应用 + 动态侧栏 + 路由
├── auto_update_windows.go  # 在线 MSI 自动更新（msiexec /i ... /qn）
├── resources/icon.svg      # 应用图标（任务栏/窗口）
├── _tools/
│   ├── GameEngineLauncher233.wxs        # WiX 源（MSI 安装包定义）
│   ├── build-msi.ps1          # 从 .wxs 构建 MSI
│   ├── install-local.ps1      # 构建 + 静默安装 + 启动
│   ├── install-msi.ps1       # 验证 SHA256 后 msiexec /qn
│   ├── release-checksums.ps1  # SHA-256 校验
│   └── publish-release.ps1    # GitHub Release 发布
├── build/windows-amd64/    # 构建产物（exe + setup.exe + msi）
└── .github/workflows/ci.yml  # CI: push main → test → build → artifact
```

## 构建产物

每次 `task build` 产出：
| 文件 | 用途 | 大小 |
|---|---|---|
| `GameEngineLauncher233.exe` | 原生可执行文件 | ~13.7 MB |
| `gameenginelauncher233-setup-<ver>-amd64.exe` | NSIS 安装程序（mygo 产） | ~4.6 MB |
| `gameenginelauncher233-<ver>-amd64.msi` | MSI 安装包（WiX 产，自动更新用） | ~5.1 MB |

## ⭐ 标准执行流程（每次处理完必走）

```powershell
# 1. 测试
task test

# 2. 构建（mygo build → NSIS setup + wix MSI）
task build

# 3. 静默安装到本机 per-user 目录并启动
task install-local

# 4. 验证安装版本（必须输出目标版本号才算成功）
& "$env:LOCALAPPDATA\Programs\GameEngineLauncher233\GameEngineLauncher233.exe" version

# 5. 提交 + push（workspace 干净）
git add -A
git commit -m "<type>: <summary>"
git push origin main
```

**安装位置**: `%LOCALAPPDATA%\Programs\GameEngineLauncher233\`
**卸载**: `msiexec /x {b3373319-8b2b-4b88-a782-725de6fb774a} /qn`（WiX UpgradeCode）

## 引擎适配器（engine_adapter*.go）

Unity 和团结引擎只是注册表里的前两个适配器。新增引擎或社区 DIY 适配都走同一套
`engineAdapter`，不要在 view 层写 `if kindUnity`。

```go
type engineAdapter struct {
    ID, Version, Description string
    Icon, BrandIcon          *ui.SVG
    Pages                    []enginePage // 每个引擎自己的导航与路由
    Features                 engineAdapterFeatures
    URLs                     engineAdapterURLs
    Process                  *engineAdapterProcess // 可选 JSON-RPC
    BuiltIn                  bool
}
```

**规则**：
- 内置适配器在 `engine_adapter.go`；用户适配器从
  `%APPDATA%\GameEngineLauncher233\engine-adapters\<id>\adapter.json` 加载。
- 视图只读 `a.engineStrategy()`：`Pages`、`Nav`、`BrandIcon`、功能标记都从
  适配器投影出来；`engineAdapterByID` / `engineAdapters` 是唯一注册表入口。
- Unity 适配器没有“游戏发行”；团结适配器有。新增页面时改对应适配器的
  `Pages`，不要改全局导航列表。
- 外部适配器页面路由为 `/engine/<id>/<page-id>`；`externalPage` 只处理
  原生 kind 的复用，其余渲染 manifest 的 links groups。
- `process` 使用 LSP 的 `Content-Length` JSON-RPC 帧，方法为
  `initialize` / `detect` / `launch` / `shutdown`；`detect` 结果会合并进扫描，
  `launch` 用于外部引擎自己的项目启动约定。
- 导入只下载 manifest，不下载/执行二进制；`adapter.json` 校验 ID、页面、
  URL、图标大小、进程相对路径和保留名称。规范与示例：
  `ENGINE_ADAPTERS.md`、`examples/engine-adapter/`。
- 适配器测试在 `engine_adapter_test.go`；改动注册表或页面路由时同时跑
  `TestEngineNavigationDiffersByAdapter`、`TestExternalAdapterDrivesRailAndPages`、
  `TestExternalAdapterProcessDetectAndLaunch`。

## 侧栏布局（垂直引擎栏）

参考 Tuanjie Hub 1.4.8 的三段式布局，视觉采用 Vercel 单色风格：

- 最左 **64px 垂直引擎栏**：注册表里的每个适配器从上往下排列；当前引擎用浮起底块标记，点击切换 `activeEngine` 与对应页面。该栏独立于导航滚动。
- 图标使用官方资源：`iconUnity` 为官方 Unity 立方体，`iconTuanjieBrand` 为官方团结引擎多面体，`iconGodotBrand` 为官方 Godot 标志；不要用自绘近似图标。
- 中间 **220px 导航列**：顶部是当前引擎品牌 + 引擎设置齿轮，中部是适配器自己的页面（32px 行高、灰阶、选中浮起），底部是许可证单行说明、下载入口和固定的 **系统设置** 入口。Godot / 链接型适配器不显示许可证或下载。
- 引擎栏与导航各自是 `ui.FocusGroup`：一个 Tab 站点，方向键遍历；快捷键 `Ctrl+1…9` 按注册表顺序切引擎，`Ctrl+,` 打开系统设置。
- 弹窗统一用 `modal()`（`ui_layout.go`），不要把 `ui.Modal` 直接用到黑色画布上，否则面板边界会消失。
- 布局回归测试在 `engine_rail_test.go`；改动侧栏时同时检查 1180×760 与 900×560。

## 品牌图标链路

```
icons.go ── iconUnity / iconTuanjie / iconTuanjieBrand / iconGodot / iconGodotBrand / iconBrand
   │                              │
   ├── 引擎栏 / 安装卡片 ── engineStrategy.BrandIcon（官方原色标志）
   ├── 侧栏品牌区 / 标题栏 ── engineStrategy.Icon / BrandIcon ── 跟随 activeEngine
   └── 主内容空状态 ── iconBrand（共享 Unity+X 标记）

resources/icon.svg ── 应用图标（任务栏/窗口）── 通用引擎启动标志（黑底白标）
```

图标来源：`D:\IDE\Tuanjie Hub\resources\app.asar` 内
`node_modules/@unity-hub-components/icon/dist/icon/assets.js` 的
`unity_logo` / `tuanjie_hub_logo` 路径；Godot 图标来自
`https://godotengine.org/press`。完整署名见 `resources/ATTRIBUTION.md`。

## MSI 自动更新链路

```
auto_update_windows.go → msiexec /i gameenginelauncher233-<ver>-amd64.msi /qn /norestart
                                    ↑
                           GitHub Release asset（sha256 校验）
```

升级规则: per-user MSI，MajorUpgrade `AllowSameVersionUpgrades="yes"`。

## 提交规范（Conventional Commits）

```
feat: 添加 xxx 功能
fix: 修复 xxx 问题
refactor: 重构 xxx（不改变行为）
chore: 构建/依赖/配置调整
```

## 验证标准

每次交付必须满足：
- `go build ./...` 零错误零警告
- `go test ./...` 全部通过
- `task build` 产出 exe + setup.exe + msi
- `task install-local` 静默安装成功，版本号匹配

## 已知约束

- Windows runner 的 `core.autocrlf` 会把 LF 变 CRLF → 仓库用 `.gitattributes` 强制 `* text=auto eol=lf`
- 引擎安装包 GB 级验证用"模拟安装"，不要真的下载
- 版本号在 `mygo.json`，MSI 版本从这里读取
