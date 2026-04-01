# ADR 0023: Agent Idle Timeout for Auto-Suspend/Auto-Resume

## Status

Accepted

## Date

2026-04-01

## Context

After ADR 0022 ("Agent Always Running"), every Agent creates a persistent Deployment. While this simplifies the mental model, it means idle Agents consume compute resources continuously. For teams with many Agents or usage patterns with long idle periods (e.g., dev Agents used only during business hours), this is wasteful.

Users already have `spec.suspend` for manual lifecycle control. However, manual suspend/resume requires human intervention and is easy to forget. Users need automatic lifecycle management — scale to zero when idle, resume when work arrives.

This is a common pattern in the Kubernetes ecosystem:
- **Knative**: `scale-to-zero-grace-period` annotation
- **KEDA**: `cooldownPeriod` + `idleReplicaCount: 0`
- **Kubernetes CronJob**: `spec.suspend` (manual only)

## Decision

Add `spec.idleTimeout` to Agent as a companion to the existing `spec.suspend`. The Agent controller handles both auto-suspend (scale to zero after idle) and auto-resume (scale back up when Tasks arrive).

### API

```yaml
spec:
  suspend: false        # Manual control (existing). Always takes priority.
  idleTimeout: "30m"    # Auto-suspend after 30min idle (new).

status:
  suspended: true
  idleSince: "2026-04-01T10:00:00Z"  # When agent became idle (new).
```

### Behavior

| `spec.suspend` | `spec.idleTimeout` | Behavior |
|---|---|---|
| `true` | any | Manual suspend. Tasks queue indefinitely. |
| `false` | nil | Always running (no auto-suspend). |
| `false` | `"30m"` | Auto-suspend after 30min idle, auto-resume on new Task. |

### Implementation

The **agent controller** handles everything. The task controller needs no changes.

1. **Auto-suspend**: Agent controller counts active Tasks (Running/Queued/Pending) on each reconcile. When count drops to 0, it records `status.idleSince`. When idle duration exceeds `idleTimeout`, it scales the Deployment to 0.

2. **Auto-resume**: Agent controller watches Tasks via label selector. When a new Task is created targeting this Agent, the watch triggers an immediate reconcile. The controller sees active tasks > 0, clears `idleSince`, and scales the Deployment back to 1.

3. **Cold start**: ~30-60 seconds for Deployment + OpenCode initialization. Mitigated by `persistence` (PVCs survive restarts).

## Consequences

### Positive

- Zero-effort resource optimization for idle Agents
- Clean API: `idleTimeout` is a single field, no complex policy objects
- No external dependencies (vs KEDA approach)
- Task controller unchanged — separation of concerns preserved
- `spec.suspend` and `idleTimeout` coexist cleanly (manual always wins)

### Negative

- Cold start latency (~30-60s) when resuming from idle
- Agent controller now watches Tasks — slightly increased reconcile load
- 30-second reconcile interval means up to 30s delay before auto-suspend (acceptable)

### Neutral

- Task watch enables near-instant auto-resume (no polling delay for new tasks)
- `status.idleSince` provides observability into idle state
