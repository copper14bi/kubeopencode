# ADR 0022: Agent Always Running — Unified Execution Model

## Status

Accepted — Supersedes [ADR 0011](0011-agent-server-mode.md)

## Date

2026-03-31

## Context

KubeOpenCode's Agent CRD currently operates in two modes:

1. **Pod mode** (default): `spec.serverConfig` is nil. Each Task creates a new ephemeral Pod that runs `opencode run` standalone. The Pod starts, executes the task, and terminates. The Agent CR itself doesn't create any running infrastructure.

2. **Server mode**: `spec.serverConfig` is set. The Agent controller creates a persistent Deployment + Service running `opencode serve`. Tasks create lightweight Pods that connect to the server via `opencode run --attach <url>`.

This dual-mode design creates several problems:

### Cognitive Overhead

Users must understand when to use each mode and how they differ. The word "Agent" means two fundamentally different things:
- In Pod mode: a static configuration document (no running resources)
- In Server mode: a running service with Deployment, Service, PVCs

This distinction is non-obvious and confusing. In common AI/ML usage, "agent" implies a running entity, not a passive config.

### API Complexity

The `ServerConfig` struct acts as a mode toggle that changes the entire behavior of the Agent controller. The presence or absence of a single field (`serverConfig`) determines:
- Whether a Deployment and Service are created
- Whether PVCs are provisioned
- Whether tasks use `--attach` or standalone execution
- Whether suspend/resume is available
- Which container image is used for task Pods (devbox ~1GB vs attach ~25MB)

### Code Complexity

The controllers contain mode-switching logic throughout:
- `agent_controller.go`: `if agentCfg.serverConfig == nil { cleanupServerResources; return }`
- `task_controller.go`: `if agentConfig.serverConfig != nil { serverURL = ... }`
- `pod_builder.go`: `if serverURL != "" { use --attach } else { use standalone }`
- `server_builder.go`: `if serverConfig == nil { return nil }`

Every new feature must consider both modes, doubling the testing surface.

### AgentTemplate Under-utilization

AgentTemplate exists as an optional base configuration for Agents. In the current model, templates provide little value beyond configuration inheritance. They could serve a more meaningful role.

## Decision

### Core Change: Agent = Always Running Instance

**Agent always creates a Deployment + Service.** There is no Pod mode vs Server mode. When you create an Agent CR, it results in a running OpenCode server.

This aligns with the intuitive meaning of "Agent" — a running entity that can receive and process tasks.

### Task Gains `templateRef`

Task spec gets a new field `templateRef` as an alternative to `agentRef`:

```yaml
# Option 1: Send to a running Agent
apiVersion: kubeopencode.io/v1alpha1
kind: Task
spec:
  agentRef:
    name: my-agent          # Agent must exist and be running
  description: "Fix the bug"

# Option 2: Run from a template (ephemeral)
apiVersion: kubeopencode.io/v1alpha1
kind: Task
spec:
  templateRef:
    name: my-template       # Creates ephemeral Pod, lifecycle = Task
  description: "Fix the bug"
```

Exactly one of `agentRef` or `templateRef` must be set (enforced by CEL validation).

### AgentTemplate = Blueprint for Ephemeral Tasks

AgentTemplate gains a new purpose: it's the blueprint for one-off task execution. When a Task references a template via `templateRef`, the Task controller creates an ephemeral Pod using the template's configuration (images, workspace, credentials, etc.). This Pod runs standalone `opencode run` and terminates when done — equivalent to the current Pod mode.

This gives AgentTemplate a clear, distinct role:
- **Agent** = running instance (persistent, accepts tasks via `--attach`)
- **AgentTemplate** = blueprint (no running resources, used for ephemeral tasks)

### API Changes

**Agent spec — `ServerConfig` removed, fields promoted:**
```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
spec:
  # These were previously nested under serverConfig:
  port: 4096                    # Server port (was serverConfig.port)
  persistence:                  # Was serverConfig.persistence
    sessions:
      size: "1Gi"
    workspace:
      size: "10Gi"
  suspend: false                # Was serverConfig.suspend

  # These remain unchanged:
  executorImage: ...
  agentImage: ...
  attachImage: ...              # Used by agentRef task Pods
  workspaceDir: /workspace
  serviceAccountName: ...
  # ... all other fields unchanged
```

**Agent status — `ServerStatus` flattened:**
```yaml
status:
  deploymentName: my-agent-server
  serviceName: my-agent
  url: http://my-agent.namespace.svc.cluster.local:4096
  ready: true
  suspended: false
  conditions: [...]
```

**Task spec — `templateRef` added:**
```yaml
spec:
  # Exactly one must be set:
  agentRef:
    name: my-agent
  # OR
  templateRef:
    name: my-template
```

### Execution Flows

**Task with `agentRef` (sends to running Agent):**
```
Task created with agentRef
  → Task controller finds Agent → gets server URL
  → Creates lightweight Pod with: opencode run --attach <url> "task"
  → Pod uses attachImage (~25MB), connects to Agent's server
  → Agent processes task, Pod reports result
```

**Task with `templateRef` (ephemeral, one-off):**
```
Task created with templateRef
  → Task controller finds AgentTemplate → builds config from template
  → Creates standalone Pod with: opencode run "task"
  → Pod uses executorImage (full devbox), runs independently
  → Pod completes, Task is done
```

**Agent lifecycle:**
```
Agent created
  → Agent controller creates Deployment (opencode serve) + Service
  → Status updated: ready=true, url=http://...
  → Agent accepts tasks via --attach

Agent suspended (spec.suspend=true)
  → Deployment scaled to 0
  → Tasks targeting this Agent enter Queued phase
  → PVCs and Service retained

Agent resumed (spec.suspend=false)
  → Deployment scaled back to 1
  → Queued tasks begin execution
```

## Consequences

### Positive

1. **Simpler mental model**: Agent = running instance, AgentTemplate = blueprint. No mode confusion.
2. **Intuitive naming**: "Agent" now means what users expect — a running entity.
3. **Less code complexity**: Remove all mode-switching conditionals from controllers.
4. **AgentTemplate has clear purpose**: Not just optional config inheritance, but the primary way to run one-off tasks.
5. **Consistent Agent behavior**: `kubectl get agents` always shows deployment status, readiness.

### Negative

1. **Breaking API change**: `serverConfig` field removed, Task `agentRef` becomes optional. All existing Agent/Task YAMLs need updating.
2. **Resource overhead for simple cases**: Users who just want a quick one-off task must now create an AgentTemplate instead of a minimal Agent. However, AgentTemplate already exists and the overhead is negligible.
3. **Every Agent consumes resources**: Even idle Agents run a Deployment. Mitigated by `suspend: true` which scales to 0.

### Neutral

1. **No migration path**: Acceptable at v1alpha1 with ~25 stars.
2. **ADR 0011 superseded**: The server mode concept is absorbed into the default Agent behavior.
3. **Template-based tasks skip concurrency/quota**: Since there's no persistent Agent to enforce limits on, `templateRef` tasks run immediately. Users who need rate limiting should use Agents.

## Alternatives Considered

### Keep dual-mode but rename

Rename Pod mode to "Ephemeral" and Server mode to "Persistent" while keeping `serverConfig`. Rejected because it doesn't reduce cognitive load — users still need to understand two modes.

### Remove Pod mode entirely

Require all tasks to target a running Agent (no `templateRef`). Rejected because some users genuinely need one-off tasks without maintaining a persistent Agent (CI/CD pipelines, batch operations).

### Create temporary Agent CRs from templates

When Task uses `templateRef`, create a temporary Agent CR. Rejected because it adds unnecessary resources (`kubectl get agents` would show ephemeral agents) and complicates cleanup.

## References

- [ADR 0011: Agent Server Mode](0011-agent-server-mode.md) (superseded)
- [ADR 0014: Remove TaskTemplate CRD](0014-remove-tasktemplate.md) (related — removed TaskTemplate in favor of Agent/AgentTemplate)
