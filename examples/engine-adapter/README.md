# Example Engine Adapter

Copy this folder to:

```
%APPDATA%\UnityStarter\engine-adapters\example-engine\
```

Then open **Settings → Engine adapters** (or restart UnityStarter). The
Example Engine appears in the left rail with its own pages and links.

To try the import flow instead, put this repository on GitHub and use:

```
https://github.com/<owner>/<repo>
```

UnityStarter reads `adapter.json` from the repository root (or a
`/tree/<branch>/<folder>` URL).

This example is declarative only. For detection and project launching, add a
JSON-RPC process and follow [`ENGINE_ADAPTERS.md`](../../ENGINE_ADAPTERS.md).
