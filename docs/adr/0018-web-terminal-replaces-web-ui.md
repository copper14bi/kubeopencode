# ADR 0018: Web Terminal Replaces OpenCode Web UI

**Status:** Accepted (Implemented)
**Date:** 2026-03-27
**Supersedes:** [ADR 0017](0017-opencode-web-ui-integration.md)

## Context

KubeOpenCode integrated the OpenCode Web UI (a SPA built with SolidJS) via a reverse proxy handler (`agent_web_handler.go`, 446 lines). This approach required:

- Downloading or building OpenCode's SPA assets and embedding them via `go:embed`
- A `bun`-based Docker build stage (`oven/bun:1.3.6-alpine`) to compile the SPA from source
- Tracking ~20 OpenCode API routes for proxy detection vs. SPA fallback routing
- JavaScript fetch monkey-patching to rewrite hardcoded `localhost:4096` URLs
- Regular version syncing with OpenCode releases

Missing a single API route caused **silent failures** (HTML returned as JSON). The maintenance burden was disproportionate to the value provided.

## Decision

Replace the OpenCode Web UI proxy with a **web terminal** using:

1. **Frontend:** xterm.js (v6.0+) terminal emulator in the browser
2. **Backend:** WebSocket endpoint that bridges to Kubernetes pod exec
3. **Execution:** `opencode attach http://localhost:{port}` runs inside the agent's server pod, providing the full OpenCode TUI experience

The architecture:
```
Browser (xterm.js) --WebSocket--> KubeOpenCode Server --pod exec--> Agent Server Pod
                                                                     $ opencode attach
```

## Consequences

### Positive

- **Eliminated bun Docker stage** — simpler, faster image builds
- **No SPA asset management** — no `go:embed`, no CDN downloads, no version pinning
- **No API route tracking** — terminal I/O is protocol-agnostic
- **Zero coupling to OpenCode internals** — uses standard Kubernetes pod exec API
- **Better TUI experience** — OpenCode's TUI is the primary interface, fully keyboard-driven
- **~380 net lines of code removed** (540 added, 920 removed)
- **Kind cluster support works out of the box** — no network access needed for assets

### Negative

- **Terminal-only UI** — no browser-native GUI, requires terminal literacy
- **Session persistence** — exec session ends when browser disconnects (mitigatable with tmux)
- **RBAC requirement** — users need `pods/exec` permission on agent server pods

### Neutral

- **Server mode only** — terminal requires a running server pod (same as previous Web UI)
- **xterm.js dependency** — added to UI bundle (~330KB additional, but removed OpenCode SPA assets)

## Files Changed

### Removed
- `internal/opencode-app/embed.go` — embedded SPA assets
- `internal/opencode-app/dist/` — SPA build output
- `internal/server/handlers/agent_web_handler.go` — 446-line reverse proxy
- `internal/server/handlers/agent_web_handler_test.go` — proxy tests
- `ui/src/components/WebUIPanel.tsx` — iframe panel component
- `docs/opencode-upgrade.md` — SOP for Web UI version upgrades
- Dockerfile `opencode-ui-builder` stage (bun-based)
- Makefile `opencode-app-build/clean` targets

### Added
- `internal/server/handlers/agent_terminal_handler.go` — WebSocket terminal handler
- `ui/src/components/TerminalPanel.tsx` — xterm.js terminal component
- `gorilla/websocket` Go dependency
- `@xterm/xterm`, `@xterm/addon-fit` npm dependencies
