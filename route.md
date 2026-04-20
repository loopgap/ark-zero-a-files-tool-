# ArkKB Architecture Baseline

## Runtime Stack
- Desktop shell: Tauri 2 (`frontend/src-tauri`)
- Frontend: React 19 + TypeScript + Vite (`frontend/src`)
- Backend/service layer: Go sidecar (`src/sidecar`, `src/bridge`, `src/core`)
- KV/config storage: bbolt
- Search/index storage: Bluge

## Top-Level Layout
```text
.
├── .github/workflows/       # CI and release workflows
├── docs/                    # help, developer notes, wiki, workspace architecture, CI/release notes
├── frontend/
│   ├── src/                 # React application and feature UI
│   ├── src-tauri/           # Tauri desktop shell
│   └── package.json         # frontend scripts
├── scripts/                 # Go-based project scripts
├── src/
│   ├── bridge/              # RPC-facing orchestration and workspace actions
│   ├── core/
│   │   ├── backup/          # archive/export logic
│   │   ├── config/          # config models and normalization
│   │   ├── file/            # file IO helpers and safe save/delete/open
│   │   ├── lsp/             # editor/LSP lifecycle
│   │   ├── render/          # markdown and preview rendering
│   │   ├── storage/         # bbolt + bluge persistence
│   │   └── sync/            # workspace indexing and sync engine
│   ├── sidecar/             # HTTP server and RPC registration
│   └── utils/               # charset/path/locking utilities
├── README.md                # project entry and release overview
├── VERSION                  # release version source
└── route.md                 # this architecture baseline
```

## Request Flow
1. React UI calls desktop RPC helpers in `frontend/src/lib`.
2. Tauri forwards commands to the Go sidecar bridge methods.
3. `src/bridge` validates workspace boundaries, coordinates storage, sync, help docs, archive browsing, and file operations.
4. `src/core/storage` persists config/membership/meta data in bbolt and search documents in Bluge.
5. `src/sidecar/server.go` exposes local HTTP endpoints for file streaming, markdown rendering, and help previews.

## Current Constraints
- RPC method names are compatibility-sensitive and should remain stable unless there is a migration plan.
- Workspace path validation must be canonical-path based; lexical prefix checks are not sufficient.
- Full-text search behavior should stay index-driven; request-time file reads are a regression.
- Embedded help docs are the source of truth for packaged builds; disk fallback is only a local-development fallback.
- Frontend async loaders must reject stale responses when archive/search/help/workspace requests race.
- Build artifacts should flow through ignored local directories, CI artifacts, and GitHub Releases rather than Git tracking.

## Active Improvement Direction
- Keep the current Tauri + React + Go sidecar split.
- Continue treating bbolt as lightweight config/meta storage and Bluge as the search/index authority.
- Prefer internal helper types and orchestration layers over RPC shape changes.
- Split `WorkbenchApp` by state domain rather than adding more responsibilities to the existing monolith.
- Keep architecture docs, workflows, and user-facing help in sync with feature work.
