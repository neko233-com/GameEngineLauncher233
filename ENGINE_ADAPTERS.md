# GameEngineLauncher233 Engine Adapter Specification

Status: **v1** (experimental, stable enough for community adapters)

GameEngineLauncher233 is an engine manager, not a game engine front-end. Built-in
adapters manage Unity and Tuanjie; every other engine is a removable adapter.
The contract is deliberately small and process-based, in the spirit of a
language server: a manifest teaches the launcher what pages an engine owns,
and an optional JSON-RPC process teaches it how to detect and launch that
engine's editors.

## 1. Model

```
Engine adapter
├── adapter.json        declarative identity, pages, links, features
├── icon.svg            optional, referenced by iconSvg or embedded inline
└── bin/<process>       optional JSON-RPC process (detect / launch)
```

An adapter owns:

- its rail icon and display name;
- its own navigation and pages;
- its own external URLs (store, docs, license, learn, community);
- optional editor/project detection and launching;
- future optional capabilities, negotiated at `initialize`.

Built-in adapters use the same registry, so the launcher has no special path
for “third-party engines”. Unity and Tuanjie are the first two entries.

## 2. Installing

### From the UI

Settings → **Engine adapters** → paste a GitHub repository, GitHub file URL,
`owner/repo`, or a direct `adapter.json` URL → **Import**.

The importer downloads only `adapter.json` (max 1 MiB), validates it, and
stores it at:

```
%APPDATA%\GameEngineLauncher233\engine-adapters\<id>\adapter.json
%APPDATA%\GameEngineLauncher233\engine-adapters\<id>\source.txt
```

A process adapter must place its executable inside that folder, or use an
absolute command. Importing never executes downloaded code.

### Locally

Copy the folder into `%APPDATA%\GameEngineLauncher233\engine-adapters\<id>\` and restart
or re-open Settings → Engine adapters. `adapter.json` must be the top file.

## 3. Manifest

```json
{
  "schemaVersion": 1,
  "id": "godot",
  "name": "Godot",
  "version": "0.1.0",
  "description": "Godot editor adapter",
  "homepage": "https://github.com/example/GameEngineLauncher233-godot",
  "iconSvg": "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 24 24\"><path fill=\"currentColor\" d=\"...\"/></svg>",
  "brandIconSvg": "<svg ...>...</svg>",
  "features": {
    "licenseFlow": false,
    "createProject": false,
    "downloads": false,
    "templates": false
  },
  "urls": {
    "docs": "https://docs.godotengine.org/",
    "community": "https://forum.godotengine.org/"
  },
  "pages": [
    {
      "id": "hub",
      "title": "Godot",
      "kind": "links",
      "icon": "projects",
      "groups": [
        {
          "title": "Godot",
          "links": [
            {"title": "Documentation", "detail": "Official manual", "url": "https://docs.godotengine.org/"}
          ]
        }
      ]
    }
  ],
  "process": {
    "command": "bin/godot-adapter.exe",
    "args": []
  }
}
```

### Fields

| field | required | notes |
|---|---|---|
| `schemaVersion` | yes | must be `1` |
| `id` | yes | lowercase slug, `[a-z][a-z0-9._-]{1,63}`, not `unity`/`tuanjie`/`system` |
| `name` | yes | 1–80 characters; shown in the rail and title |
| `version` | no | defaults to `0.1.0` |
| `description` | no | shown in Settings |
| `homepage` | no | http(s) URL |
| `iconSvg` | no | monochrome SVG; `currentColor` recommended |
| `brandIconSvg` | no | full-colour SVG; used on the active rail tab |
| `features` | no | capability flags; see below |
| `urls` | no | engine-owned destinations used by shared pages |
| `pages` | yes | 1–50 pages; see below |
| `process` | no | LSP-like JSON-RPC process |

`features`:

- `licenseFlow` — the engine needs the Unity license workspace (default false).
- `createProject` — the process can create projects (reserved; v1 UI only
  enables new-project actions for built-ins).
- `downloads` — the adapter has a downloads workspace; without it the pinned
  Downloads entry is hidden.
- `templates` — reserved for a future template protocol.

### Pages

`pages[].kind` is one of:

- `projects`, `installs`, `favorites`, `activity` — native workspaces scoped to
  the adapter's editors/projects;
- `downloads` — native downloads workspace (only for engines with the flag);
- `links` — a manifest-defined page of link groups (recommended for custom
  engines);
- `publish`, `license`, `learn`, `docs`, `store`, `feedback` — native
  link-oriented pages. For user adapters they render the page's own groups.

Routes are generated as `/engine/<adapter-id>/<page-id>`. Built-in Unity and
Tuanjie keep their historical routes for compatibility. A page with
`kind: "projects"` becomes the landing page; otherwise the first non-download
page is the landing page.

`pages[].icon` accepts: `projects`, `installs`, `download`, `favorite`,
`learn`, `store`, `publish`, `license`, `docs`, `feedback`, or any value
falls back to the generic engine mark.

`groups[].links[].url` must be http(s). Links open in the system browser.

## 4. Process protocol (v1)

If `process` is present, the launcher starts:

```
<adapter-dir>/<command> <args...>
```

with `cwd = <adapter-dir>` and speaks **LSP framing**:

```
Content-Length: <bytes>\r\n
\r\n
<JSON-RPC 2.0 message>
```

One JSON-RPC request per object. Messages without a matching ID are ignored
(notifications/logs are allowed). The maximum message size is 4 MiB.

### `initialize`

Request:

```json
{
  "jsonrpc": "2.0", "id": 1, "method": "initialize",
  "params": {
    "protocolVersion": 1,
    "client": {"name": "GameEngineLauncher233", "version": "1.9.0"},
    "engine": {"id": "godot", "name": "Godot"}
  }
}
```

Result (capabilities are advisory in v1):

```json
{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"detect":true,"launch":true}}}
```

### `detect`

Called during a scan. It should be fast and side-effect free.

```json
{"jsonrpc":"2.0","id":2,"method":"detect","params":{}}
```

Result:

```json
{
  "editors": [
    {"version":"4.3.0","displayVersion":"4.3","exe":"C:\\Godot\\godot.exe","root":"C:\\Godot"}
  ],
  "projects": [
    {"path":"C:\\Games\\demo","name":"demo","version":"4.3.0","modified":"2026-01-05T12:00:00Z"}
  ]
}
```

The launcher adds these to the Installs and Projects workspaces. `modified`
is RFC 3339; malformed values are ignored.

### `launch`

Called when the user opens a project through the adapter, or launches the
editor from its install card.

```json
{
  "jsonrpc":"2.0","id":3,"method":"launch",
  "params":{"exe":"C:\\Godot\\godot.exe","projectPath":"C:\\Games\\demo","args":[]}
}
```

Return `{"ok":true}` or a JSON-RPC error. The adapter owns its CLI
conventions; the launcher never guesses flags for an external engine.

### `shutdown`

Called before the launcher closes stdin. Return any object, then exit.

### Timeouts

The launcher gives each `initialize`/`detect` 6 seconds and `launch` 8
seconds. A timeout or crash is isolated to that adapter; built-in engines and
other adapters keep working.

## 5. Versioning and compatibility

- `schemaVersion` is the manifest format. Unknown versions are refused.
- `protocolVersion` is the process protocol. v1 has no required capability
  negotiation beyond starting and answering known methods.
- Future additions will be additive: new optional fields, new methods, and
  capability flags. Adapters must ignore unknown JSON fields.
- The launcher must never fail because one external adapter is incomplete.

## 6. Security and trust

- Import only from sources you trust. A process adapter is native code and
  runs with your user account.
- The importer downloads a manifest only; it does not fetch or execute
  binaries.
- Manifest IDs are slugs, page IDs are slugs, process commands are confined to
  the adapter folder when relative, and URLs must be http(s).
- Built-in adapters cannot be removed or replaced.

## 7. Reference implementation

`examples/engine-adapter/` is a copyable declarative adapter. The native tests
also exercise:

- a manifest-only adapter (`TestExternalAdapterDrivesRailAndPages`);
- a real stdio JSON-RPC process (`TestExternalAdapterProcessDetectAndLaunch`);
- import/validation/removal (`TestImportAdapterFromURLAndRemove`,
  `TestAdapterManifestRejectsUnsafeValues`);
- per-engine navigation (`TestEngineNavigationDiffersByAdapter`).

The source of truth for the parser and protocol is:

- `engine_adapter_manifest.go`
- `engine_adapter_install.go`
- `engine_adapter_process.go`
- `engine_adapter_ui.go`
