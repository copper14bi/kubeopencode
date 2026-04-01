# KubeOpenCode Architecture & API Design

## Table of Contents

1. [System Overview](#system-overview)
2. [API Design](#api-design)
3. [System Architecture](#system-architecture)
4. [Custom Resource Definitions](#custom-resource-definitions)
5. [Agent Configuration](#agent-configuration)
6. [System Configuration](#system-configuration)
7. [Complete Examples](#complete-examples)
8. [kubectl Usage](#kubectl-usage)
9. [Web UI](#web-ui)

---

## System Overview

KubeOpenCode brings Agentic AI capabilities into the Kubernetes ecosystem. By leveraging Kubernetes, it enables AI agents to be deployed as services, run in isolated virtual environments, and integrate with enterprise management and governance frameworks.

### Core Goals

- Use Kubernetes CRDs to define Task and Agent resources
- Use Controller pattern to manage resource lifecycle
- Agents run as persistent Deployments; Tasks execute via lightweight attach Pods or ephemeral template Pods
- Seamless integration with Kubernetes ecosystem

### Key Advantages

- **Native Integration**: Works seamlessly with Helm, Kustomize, ArgoCD and other K8s tools
- **Declarative Management**: Use K8s resource definitions, supports GitOps
- **Infrastructure Reuse**: Logs, monitoring, auth/authz all leverage K8s capabilities
- **Simplified Operations**: Manage with standard K8s tools (kubectl, dashboard)
- **Batch Operations**: Use Helm/Kustomize to create multiple Tasks (Kubernetes-native approach)

### External Integrations

KubeOpenCode focuses on the core Task/Agent abstraction. For advanced features, integrate with external projects:

| Feature | Recommended Integration |
|---------|------------------------|
| Workflow orchestration | [Argo Workflows](https://argoproj.github.io/argo-workflows/) |
| Event-driven triggers | [Argo Events](https://argoproj.github.io/argo-events/) |
| Scheduled execution | Kubernetes CronJob |

See the [kubeopencode/dogfooding](https://github.com/kubeopencode/dogfooding) repository for examples of GitHub webhook integration using Argo Events that creates KubeOpenCode Tasks.

---

## API Design

### Resource Overview

| Resource | Purpose | Stability |
|----------|---------|-----------|
| **Task** | Single task execution (primary API) | Stable - semantic name |
| **Agent** | Running AI agent instance (Deployment + Service) | Stable - independent of project name |
| **AgentTemplate** | Reusable blueprint for Agents and ephemeral Tasks | Stable - configuration inheritance |
| **KubeOpenCodeConfig** | Cluster-scoped system-level configuration | Stable - system settings |
| **ContextItem** | Inline context for AI agents (KNOW) | Stable - inline context only |

### Key Design Decisions

#### 1. Task as Primary API

**Rationale**: Simple, focused API for single task execution. For batch operations, use Helm/Kustomize to create multiple Tasks.

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Task
```

#### 2. Agent (not KubeOpenCodeConfig)

**Rationale**:
- **Stable**: Independent of project name - won't change even if project renames
- **Semantic**: Reflects architecture philosophy: "Agent = AI + permissions + tools"
- **Clear**: Configures the agent environment for task execution

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
```

#### 3. No Batch/BatchRun

**Rationale**: Kubernetes-native approach - use Helm, Kustomize, or other templating tools to create multiple Tasks. This:
- Reduces API complexity
- Leverages existing Kubernetes tooling
- Follows cloud-native best practices

#### 4. No Retry Mechanism

**Rationale**: AI tasks are fundamentally different from traditional functions:

- **Non-deterministic output**: AI agents may produce different results on each run
- **Non-idempotent operations**: Tasks may perform actions (create PRs, modify files, send messages) that should not be repeated
- **Compound failures**: Retrying a partially completed task may cause duplicate operations or inconsistent state

**Implementation**:
- Pods are created with `RestartPolicy: Never` (no retry on failure)
- Pods use `restartPolicy: Never` (no container restart on failure)
- Task fails immediately when the agent container exits with non-zero code

**If retry is needed**, use external Kubernetes ecosystem components:
- **Argo Workflows**: DAG-based workflow with conditional retry logic
- **Tekton Pipelines**: CI/CD pipelines with result-based retry
- **Custom controllers**: Monitor Task status and create new Tasks based on validation results

### Resource Hierarchy

```
Task (single task execution)
├── TaskSpec
│   ├── description: *string                (syntactic sugar for /workspace/task.md)
│   ├── contexts: []ContextItem             (inline context definitions)
│   ├── agentRef: *AgentReference           (Agent reference, same namespace)
│   └── templateRef: *AgentTemplateReference (AgentTemplate reference, alternative to agentRef)
└── TaskExecutionStatus
    ├── phase: TaskPhase
    ├── podName: string
    ├── startTime: Time
    ├── completionTime: Time
    └── conditions: []Condition

Agent (running AI agent instance — always creates Deployment + Service)
└── AgentSpec
    ├── profile: string             (brief human-readable summary, for documentation/discovery)
    ├── agentImage: string           (OpenCode init container image)
    ├── executorImage: string        (Main worker container image)
    ├── attachImage: string          (lightweight image for --attach Pods)
    ├── workspaceDir: string         (default: "/workspace")
    ├── command: []string
    ├── port: int32                  (OpenCode server port, default: 4096)
    ├── persistence: *PersistenceConfig  (session/workspace PVCs)
    ├── suspend: bool                (scale Deployment to 0 replicas)
    ├── contexts: []ContextItem      (inline context definitions)
    ├── credentials: []Credential
    ├── caBundle: *CABundleConfig    (custom CA certificates for TLS)
    ├── proxy: *ProxyConfig          (HTTP/HTTPS proxy settings)
    │   ├── httpProxy: string        (HTTP proxy URL)
    │   ├── httpsProxy: string       (HTTPS proxy URL)
    │   └── noProxy: string          (comma-separated bypass list)
    ├── imagePullSecrets: []LocalObjectReference  (private registry auth)
    ├── podSpec: *AgentPodSpec
    ├── serviceAccountName: string
    ├── maxConcurrentTasks: *int32   (limit concurrent Tasks, nil/0 = unlimited)
    └── quota: *QuotaConfig          (rate limiting for Task starts)
        ├── maxTaskStarts: int32     (max starts within window)
        └── windowSeconds: int32     (sliding window duration in seconds)

AgentTemplate (reusable blueprint for Agents and ephemeral Tasks)
└── AgentTemplateSpec
    ├── (shares most fields with AgentSpec, except profile/port/persistence/suspend)
    └── (see AgentTemplate section below for full spec)

KubeOpenCodeConfig (system configuration)
└── KubeOpenCodeConfigSpec
    ├── systemImage: *SystemImageConfig       (internal KubeOpenCode components)
    │   ├── image: string                     (default: DefaultKubeOpenCodeImage)
    │   └── imagePullPolicy: PullPolicy       (default: IfNotPresent)
    ├── cleanup: *CleanupConfig               (Task cleanup policies)
    │   ├── ttlSecondsAfterFinished: *int32   (TTL for finished Tasks, nil = disabled)
    │   └── maxRetainedTasks: *int32          (max Tasks to retain, nil = unlimited)
    └── proxy: *ProxyConfig                   (cluster-wide HTTP/HTTPS proxy)
        ├── httpProxy: string                 (HTTP proxy URL)
        ├── httpsProxy: string                (HTTPS proxy URL)
        └── noProxy: string                   (comma-separated bypass list)
```

### Complete Type Definitions

```go
// Task represents a single task execution
type Task struct {
    Spec   TaskSpec
    Status TaskExecutionStatus
}

type TaskSpec struct {
    Description *string                 // Syntactic sugar for /workspace/task.md
    Contexts    []ContextItem           // Inline context definitions
    AgentRef    *AgentReference         // Agent reference (same namespace)
    TemplateRef *AgentTemplateReference // AgentTemplate reference (alternative to agentRef)
    // Exactly one of AgentRef or TemplateRef must be set
}

// AgentReference references an Agent in the same namespace
type AgentReference struct {
    Name string // Agent name (required)
}

// ContextItem defines inline context content
type ContextItem struct {
    Type      ContextType       // Text, ConfigMap, Git, or Runtime
    MountPath string            // Empty = write to .kubeopencode/context.md (ignored for Runtime)
    FileMode  *int32            // Optional file permission mode (e.g., 0755 for executable)
    Text      string            // Content when Type is Text
    ConfigMap *ConfigMapContext // ConfigMap when Type is ConfigMap
    Git       *GitContext       // Git repo when Type is Git
    Runtime   *RuntimeContext   // Platform awareness when Type is Runtime
}

type TaskExecutionStatus struct {
    Phase          TaskPhase
    PodName        string
    StartTime      *metav1.Time
    CompletionTime *metav1.Time
    Conditions     []metav1.Condition
}

type ContextType string
const (
    ContextTypeText      ContextType = "Text"
    ContextTypeConfigMap ContextType = "ConfigMap"
    ContextTypeGit       ContextType = "Git"
    ContextTypeRuntime   ContextType = "Runtime"
)

// RuntimeContext enables KubeOpenCode platform awareness for agents.
// When enabled, the controller injects a system prompt explaining:
// - The agent is running in a Kubernetes environment as a KubeOpenCode Task
// - Available environment variables (TASK_NAME, TASK_NAMESPACE, WORKSPACE_DIR)
// - How to query Task information via kubectl
type RuntimeContext struct {
    // No fields - content is generated by the controller
}

type ConfigMapContext struct {
    Name     string // Name of the ConfigMap
    Key      string // Optional: specific key to mount
    Optional *bool  // Whether the ConfigMap must exist
}

type GitContext struct {
    Repository string              // Git repository URL
    Path       string              // Path within the repository
    Ref        string              // Branch, tag, or commit SHA (default: "HEAD")
    Depth      *int                // Shallow clone depth (default: 1)
    SecretRef  *GitSecretReference // Optional Git credentials
}

// Agent defines the AI agent configuration
type Agent struct {
    Spec AgentSpec
}

type AgentSpec struct {
    Profile            string                      // Brief human-readable summary of Agent's purpose (optional, for documentation/discovery)
    AgentImage         string                      // OpenCode init container image (copies binary to /tools)
    ExecutorImage      string                      // Main worker container image (runs tasks)
    AttachImage        string                      // Lightweight image for --attach Pods (~25MB)
    WorkspaceDir       string                      // Working directory (default: "/workspace")
    Command            []string                    // Custom entrypoint command
    Port               int32                       // OpenCode server port (default: 4096)
    Persistence        *PersistenceConfig           // Session/workspace PVC configuration
    Suspend            bool                        // Scale Deployment to 0 replicas
    Contexts           []ContextItem               // Inline context definitions
    Credentials        []Credential
    CABundle           *CABundleConfig              // Custom CA certificates for private HTTPS/Git servers
    Proxy              *ProxyConfig                 // HTTP/HTTPS proxy settings for all containers
    ImagePullSecrets   []corev1.LocalObjectReference // Private registry image pull secrets
    PodSpec            *AgentPodSpec                // Pod configuration (labels, scheduling, runtime, security)
    ServiceAccountName string
    MaxConcurrentTasks *int32                       // Limit concurrent Tasks (nil/0 = unlimited)
}

// ProxyConfig configures HTTP/HTTPS proxy for all containers in generated Pods
type ProxyConfig struct {
    HttpProxy  string // HTTP proxy URL (sets HTTP_PROXY and http_proxy)
    HttpsProxy string // HTTPS proxy URL (sets HTTPS_PROXY and https_proxy)
    NoProxy    string // Comma-separated bypass list (.svc,.cluster.local always appended)
}

// CABundleConfig configures custom CA certificates for TLS
type CABundleConfig struct {
    ConfigMapRef *CABundleReference // CA bundle from ConfigMap (default key: "ca-bundle.crt")
    SecretRef    *CABundleReference // CA bundle from Secret (default key: "ca.crt")
}

// CABundleReference references a ConfigMap or Secret containing CA certificates
type CABundleReference struct {
    Name string // Name of the ConfigMap or Secret
    Key  string // Key within the resource (optional, has type-specific default)
}

// KubeOpenCodeConfig defines system-level configuration
type KubeOpenCodeConfig struct {
    Spec KubeOpenCodeConfigSpec
}

type KubeOpenCodeConfigSpec struct {
    SystemImage *SystemImageConfig // System image for internal components
    Cleanup     *CleanupConfig     // Task cleanup configuration
    Proxy       *ProxyConfig       // Cluster-wide HTTP/HTTPS proxy settings
}

// SystemImageConfig configures the KubeOpenCode system image
type SystemImageConfig struct {
    Image           string            // System image (default: built-in DefaultKubeOpenCodeImage)
    ImagePullPolicy corev1.PullPolicy // Pull policy: Always/Never/IfNotPresent (default: IfNotPresent)
}

// CleanupConfig defines cleanup policies for completed/failed Tasks
type CleanupConfig struct {
    TTLSecondsAfterFinished *int32 // TTL for cleaning up finished Tasks (nil = disabled)
    MaxRetainedTasks        *int32 // Max completed Tasks to retain per namespace (nil = unlimited)
}
```

---

## System Architecture

### Component Layers

```
┌─────────────────────────────────────────────────────────────┐
│                   Kubernetes API Server                     │
│  - Custom Resource Definitions (CRDs)                       │
│  - RBAC & Authentication                                    │
│  - Event System                                             │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              KubeOpenCode Controller (Operator)                 │
│  - Watch Task CRs                                           │
│  - Reconcile loop                                           │
│  - Create Kubernetes Pods for tasks                         │
│  - Update CR status fields                                  │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                   Kubernetes Pods                      │
│  - Each task runs as a separate Pod                     │
│  - Execute task using agent container                       │
│  - AI agent invocation                                      │
│  - Context files mounted as volumes                         │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                      Storage Layer                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ etcd (Kubernetes Backend)                            │   │
│  │  - Task CRs                                          │   │
│  │  - Agent CRs                                         │   │
│  │  - CR status (execution state, results)              │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ ConfigMaps                                           │   │
│  │  - Task context files                                │   │
│  │  - Configuration data                                │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Custom Resource Definitions

### Task (Primary API)

Task is the primary API for executing AI-powered tasks.

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: update-service-a
  namespace: kubeopencode-system
spec:
  # Simple task description (syntactic sugar for /workspace/task.md)
  description: |
    Update dependencies to latest versions.
    Run tests and create PR.

  # Inline context definitions
  contexts:
    - type: Text
      mountPath: /workspace/guides/standards.md
      text: |
        # Coding Standards
        - Use descriptive variable names
        - Write unit tests for all functions
    - type: ConfigMap
      configMap:
        name: security-policy
      # Empty mountPath = write to .kubeopencode/context.md with XML tags

  # Required: Reference to Agent or AgentTemplate (exactly one)
  agentRef:
    name: my-agent

status:
  # Execution phase
  phase: Running  # Pending|Queued|Running|Completed|Failed

  # Kubernetes Pod name
  podName: update-service-a-xyz123

  # Start and end times
  startTime: "2025-01-18T10:00:00Z"
  completionTime: "2025-01-18T10:05:00Z"
```

**Field Description:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spec.description` | String | No | Task instruction (creates /workspace/task.md) |
| `spec.contexts` | []ContextItem | No | Inline context definitions (see below) |
| `spec.agentRef` | *AgentReference | No | Agent reference, same namespace. Exactly one of `agentRef` or `templateRef` must be set. |
| `spec.templateRef` | *AgentTemplateReference | No | AgentTemplate reference for ephemeral execution. Exactly one of `agentRef` or `templateRef` must be set. |

**Status Field Description:**

| Field | Type | Description |
|-------|------|-------------|
| `status.phase` | TaskPhase | Execution phase: Pending\|Queued\|Running\|Completed\|Failed |
| `status.podName` | String | Kubernetes Pod name |
| `status.startTime` | Timestamp | Start time |
| `status.completionTime` | Timestamp | End time |

**ContextItem Types:**

Contexts are defined inline in Task or Agent specs:

1. **Text Context** - Inline text content:
```yaml
contexts:
  - type: Text
    mountPath: /workspace/guides/standards.md  # Optional
    text: |
      # Coding Standards
      - Use descriptive variable names
```

2. **ConfigMap Context** - Content from ConfigMap:
```yaml
contexts:
  - type: ConfigMap
    mountPath: /workspace/configs  # Optional
    configMap:
      name: my-configs
      key: config.md  # Optional: specific key
```

3. **Git Context** - Content from Git repository:
```yaml
contexts:
  - type: Git
    mountPath: /workspace/repo
    git:
      repository: https://github.com/org/contexts
      path: guides/
      ref: main
```

4. **Runtime Context** - KubeOpenCode platform awareness:
```yaml
contexts:
  - type: Runtime
    runtime: {}  # No fields - content is generated by controller
```

### Context System

Contexts provide additional information to AI agents during task execution. They are defined inline in Task or Agent specs using the `ContextItem` structure.

**Context Types:**

| Type | Description |
|------|-------------|
| `Text` | Inline text content directly in YAML |
| `ConfigMap` | Content from a Kubernetes ConfigMap |
| `Git` | Content cloned from a Git repository |
| `Runtime` | KubeOpenCode platform awareness (auto-generated by controller) |
| `URL` | Content fetched from a remote HTTP/HTTPS URL at task execution time |

**ContextItem Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | Optional identifier for logging, debugging, and XML tag generation |
| `description` | string | No | Human-readable documentation for the context |
| `optional` | *bool | No | If true, task proceeds even if context cannot be resolved |
| `type` | ContextType | Yes | Type of context: Text, ConfigMap, Git, Runtime, or URL |
| `mountPath` | string | No | Where to mount (empty = write to .kubeopencode/context.md) |
| `fileMode` | *int32 | No | File permission mode (e.g., 0755 for executables) |
| `text` | string | When type=Text | Text content |
| `configMap` | ConfigMapContext | When type=ConfigMap | Reference to ConfigMap |
| `git` | GitContext | When type=Git | Content from Git repository |
| `runtime` | RuntimeContext | When type=Runtime | Platform awareness (no fields - content is generated by controller) |
| `url` | URLContext | When type=URL | Remote URL to fetch content from |

**Important Notes:**

- **Empty MountPath behavior**: When mountPath is empty, content is written to `${WORKSPACE_DIR}/.kubeopencode/context.md` with XML tags. OpenCode loads this via `OPENCODE_CONFIG_CONTENT` env var, avoiding conflicts with repository's `AGENTS.md`
- **Runtime context**: Provides KubeOpenCode platform awareness to agents, explaining environment variables, kubectl commands, and system concepts
- **Path resolution**: Relative paths are prefixed with workspaceDir; absolute paths are used as-is
- **URL context**: Fetches content at task execution time via an init container. Requires `mountPath` to be specified

**Context Priority (lowest to highest):**

1. Agent.contexts (array order)
2. Task.contexts (array order)
3. Task.description (becomes /workspace/task.md)

### Agent (Running AI Agent Instance)

Agent defines a running AI agent. Creating an Agent always results in a Deployment + Service running an OpenCode server. Tasks reference Agents via `agentRef` and execute on the persistent server.

KubeOpenCode uses a **two-container pattern** for the Agent's Deployment:
1. **Init Container** (OpenCode image): Copies OpenCode binary to `/tools` shared volume
2. **Worker Container** (Executor image): Runs the OpenCode server using `/tools/opencode serve`

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: my-agent
  namespace: kubeopencode-system
spec:
  profile: "Full-stack development agent with GitHub and AWS access"

  # OpenCode init container image (optional, has default)
  agentImage: quay.io/kubeopencode/kubeopencode-agent-opencode:latest

  # Executor container image (worker that runs OpenCode)
  executorImage: quay.io/kubeopencode/kubeopencode-agent-devbox:latest

  # Optional: Working directory (default: "/workspace")
  workspaceDir: /workspace

  # Optional: Custom entrypoint command (uses OpenCode from /tools)
  command: ["sh", "-c", "/tools/opencode run --format json \"$(cat /workspace/task.md)\""]

  # Optional: Inline contexts (applied to all tasks using this agent)
  contexts:
    - type: Text
      text: |
        # Coding Standards
        - Use descriptive variable names
        - Write unit tests for all functions
    - type: ConfigMap
      configMap:
        name: org-security-policy

  # Optional: Credentials (secrets as env vars or file mounts)
  credentials:
    # Mount entire secret as environment variables (all keys become env vars)
    - name: api-keys
      secretRef:
        name: api-credentials
        # No key specified - all secret keys become ENV vars with same names

    # Mount single key with custom env name
    - name: github-token
      secretRef:
        name: github-creds
        key: token
      env: GITHUB_TOKEN

    # Mount single key as file
    - name: ssh-key
      secretRef:
        name: ssh-keys
        key: id_rsa
      mountPath: /home/agent/.ssh/id_rsa
      fileMode: 0400

  # Optional: Advanced Pod configuration
  podSpec:
    # Labels for NetworkPolicy, monitoring, etc.
    labels:
      network-policy: agent-restricted

    # Scheduling constraints
    scheduling:
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
        - key: "dedicated"
          operator: "Equal"
          value: "ai-workload"
          effect: "NoSchedule"

    # RuntimeClass for enhanced isolation (gVisor, Kata, etc.)
    runtimeClassName: gvisor

  # Optional: Limit concurrent Tasks using this Agent
  maxConcurrentTasks: 3

  # Required: ServiceAccount for agent pods
  serviceAccountName: kubeopencode-agent
```

**Field Description:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spec.agentImage` | String | No | OpenCode init container image (copies binary to /tools) |
| `spec.executorImage` | String | No | Main worker container image (runs the server) |
| `spec.attachImage` | String | No | Lightweight image for --attach Pods (~25MB) |
| `spec.workspaceDir` | String | No | Working directory (default: "/workspace") |
| `spec.command` | []String | No | Custom entrypoint command |
| `spec.port` | int32 | No | OpenCode server port (default: 4096) |
| `spec.persistence` | *PersistenceConfig | No | Persistent storage for sessions and workspace |
| `spec.persistence.sessions` | *VolumePersistence | No | Session data PVC (SQLite DB) |
| `spec.persistence.workspace` | *VolumePersistence | No | Workspace directory PVC |
| `spec.suspend` | bool | No | Scale Deployment to 0 replicas (default: false) |
| `spec.contexts` | []ContextItem | No | Inline contexts (applied to all tasks) |
| `spec.credentials` | []Credential | No | Secrets as env vars or file mounts |
| `spec.caBundle` | *CABundleConfig | No | Custom CA certificates for private HTTPS/Git servers |
| `spec.caBundle.configMapRef.name` | string | Yes (if configMapRef) | ConfigMap name containing the CA bundle |
| `spec.caBundle.configMapRef.key` | string | No | Key in ConfigMap (default: "ca-bundle.crt") |
| `spec.caBundle.secretRef.name` | string | Yes (if secretRef) | Secret name containing the CA bundle |
| `spec.caBundle.secretRef.key` | string | No | Key in Secret (default: "ca.crt") |
| `spec.proxy` | *ProxyConfig | No | HTTP/HTTPS proxy settings for all containers |
| `spec.proxy.httpProxy` | string | No | HTTP proxy URL (sets HTTP_PROXY and http_proxy) |
| `spec.proxy.httpsProxy` | string | No | HTTPS proxy URL (sets HTTPS_PROXY and https_proxy) |
| `spec.proxy.noProxy` | string | No | Comma-separated bypass list (.svc,.cluster.local always appended) |
| `spec.imagePullSecrets` | []LocalObjectReference | No | Private registry image pull secrets (kubernetes.io/dockerconfigjson type) |
| `spec.podSpec` | *AgentPodSpec | No | Advanced Pod configuration (labels, scheduling, runtimeClass, security) |
| `spec.podSpec.securityContext` | *SecurityContext | No | Container-level security context override (default: restricted) |
| `spec.podSpec.podSecurityContext` | *PodSecurityContext | No | Pod-level security attributes (runAsUser, fsGroup, etc.) |
| `spec.maxConcurrentTasks` | *int32 | No | Limit concurrent Tasks (nil/0 = unlimited) |
| `spec.quota` | *QuotaConfig | No | Rate limiting for Task starts |
| `spec.quota.maxTaskStarts` | int32 | Yes (if quota set) | Maximum Task starts within the window |
| `spec.quota.windowSeconds` | int32 | Yes (if quota set) | Sliding window duration in seconds (60-86400) |
| `spec.serviceAccountName` | String | Yes | ServiceAccount for agent pods |

**Task Stop:**

Running Tasks can be stopped by setting the `kubeopencode.io/stop=true` annotation:

```bash
kubectl annotate task my-task kubeopencode.io/stop=true
```

When this annotation is detected:
- The controller deletes the Pod (with graceful termination)
- Kubernetes sends SIGTERM to all running Pods, triggering graceful shutdown
- Pod is deleted after termination (logs are not preserved)
- Task status is set to `Completed` with a `Stopped` condition

---

## AgentTemplate (Reusable Blueprint)

AgentTemplate serves two roles:

1. **Base configuration for Agents**: Teams maintain shared settings (images, contexts, credentials) in one template. Individual Agents reference it via `templateRef`.
2. **Blueprint for ephemeral Tasks**: Tasks can reference a template directly via `templateRef` to run one-off Pods without a persistent Agent.

### AgentTemplateSpec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `agentImage` | string | No | OpenCode init container image |
| `executorImage` | string | No | Worker container image |
| `attachImage` | string | No | Lightweight image for agentRef task Pods |
| `workspaceDir` | string | Yes | Working directory inside container |
| `command` | []string | No | Entrypoint command override |
| `contexts` | []ContextItem | No | Default contexts |
| `config` | *string | No | OpenCode JSON config |
| `credentials` | []Credential | No | Secret references |
| `podSpec` | *AgentPodSpec | No | Advanced pod configuration |
| `serviceAccountName` | string | Yes | Kubernetes ServiceAccount |
| `caBundle` | *CABundleConfig | No | Custom CA certificates |
| `proxy` | *ProxyConfig | No | HTTP/HTTPS proxy settings |
| `imagePullSecrets` | []LocalObjectReference | No | Private registry secrets |
| `maxConcurrentTasks` | *int32 | No | Default concurrency limit |
| `quota` | *QuotaConfig | No | Default rate limiting |

### Merge Strategy

When an Agent references a template via `spec.templateRef`, fields are merged:

- **Scalar/pointer fields**: Agent value wins if non-zero/non-nil; otherwise template value is used
- **List fields** (contexts, credentials, imagePullSecrets): Agent **replaces** the template list entirely if specified (not appended)
- **Agent-only fields** (profile, port, persistence, suspend): Always from Agent (not in template)

### Tracking

Agents that reference a template automatically get the label:
```
kubeopencode.io/agent-template: <template-name>
```

This enables:
- Querying all Agents using a specific template
- UI showing bidirectional links between Agents and templates
- Re-reconciliation of Agents when a template changes

### Example

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: AgentTemplate
metadata:
  name: team-python-dev
  namespace: engineering
spec:
  executorImage: quay.io/kubeopencode/kubeopencode-agent-devbox:latest
  workspaceDir: /workspace
  serviceAccountName: kubeopencode-agent
  contexts:
    - name: coding-standards
      type: Text
      text: |
        Always follow PEP 8. Use type hints.
  credentials:
    - name: github-token
      secretRef:
        name: shared-github-creds
        key: token
      env: GITHUB_TOKEN
  maxConcurrentTasks: 3          # Default concurrency for all derived Agents
  quota:
    maxTaskStarts: 50
    windowSeconds: 3600          # 50 tasks/hour default
---
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: alice-agent
  namespace: engineering
spec:
  templateRef:
    name: team-python-dev
  profile: "Alice's development agent with persistence"
  workspaceDir: /workspace
  serviceAccountName: kubeopencode-agent
  maxConcurrentTasks: 10         # Override: higher concurrency
  persistence:
    sessions:
      size: "2Gi"
---
# Ephemeral task using the template directly (no Agent needed)
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: batch-lint
  namespace: engineering
spec:
  templateRef:
    name: team-python-dev
  description: "Run linting on all Python files"
```

---

## Agent Configuration

### Agent Image Discovery

KubeOpenCode uses a **two-container pattern** for the Agent's Deployment:

1. **Init Container** (`agentImage`): Copies OpenCode binary to `/tools` shared volume
2. **Worker Container** (`executorImage`): Runs the OpenCode server using `/tools/opencode serve`

Tasks referencing an Agent via `agentRef` create lightweight Pods using the `attachImage` (~25MB) that connect to the Agent's server via `opencode run --attach <url>`.

Tasks referencing an AgentTemplate via `templateRef` create standalone Pods using the `executorImage` with the full development environment.

### Image Resolution

| Field | Container | Default |
|-------|-----------|---------|
| `agentImage` | Init Container (OpenCode) | `quay.io/kubeopencode/kubeopencode-agent-opencode:latest` |
| `executorImage` | Worker Container (Server) | `quay.io/kubeopencode/kubeopencode-agent-devbox:latest` |
| `attachImage` | Task Pod (agentRef) | `quay.io/kubeopencode/kubeopencode-agent-attach:latest` |

**Backward Compatibility:**
- If only `agentImage` is set (legacy): it's used as the executor image, default OpenCode image is used for init container
- If both are set: `agentImage` for init container, `executorImage` for worker container

### How It Works

**Agent creation:**
1. Agent controller creates a Deployment running `opencode serve` + a ClusterIP Service
2. Status is updated with deployment name, service name, URL, and readiness

**Task with `agentRef`:**
1. Task controller finds the Agent and gets its server URL
2. Creates a lightweight Pod using `attachImage` with command: `opencode run --attach <url> "task"`
3. Pod connects to the Agent's server, executes the task, and terminates

**Task with `templateRef`:**
1. Task controller finds the AgentTemplate and builds configuration from it
2. Creates a standalone Pod using `executorImage` with command: `opencode run "task"`
3. Pod runs independently with the full development environment and terminates when done

### Concurrency Control

Agents can limit the number of concurrent Tasks to prevent overwhelming backend AI services with rate limits:

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: opencode-agent
spec:
  profile: "General-purpose OpenCode agent with concurrency limits"
  # OpenCode init container (optional, has default)
  agentImage: quay.io/kubeopencode/kubeopencode-agent-opencode:latest
  # Executor container (optional, has default)
  executorImage: quay.io/kubeopencode/kubeopencode-agent-devbox:latest
  command: ["sh", "-c", "/tools/opencode run --format json \"$(cat /workspace/task.md)\""]
  serviceAccountName: kubeopencode-agent
  maxConcurrentTasks: 3  # Only 3 Tasks can run concurrently
```

**Behavior:**

| `maxConcurrentTasks` Value | Behavior |
|---------------------------|----------|
| `nil` (not set) | Unlimited - all Tasks run immediately |
| `0` | Unlimited - same as nil |
| `> 0` | Limited - Tasks queue when at capacity |

**Task Lifecycle with Queuing:**

```
Task Created
    │
    ├─── Agent has capacity ──► Phase: Running ──► Phase: Completed/Failed
    │
    └─── Agent at capacity ──► Phase: Queued
                                    │
                                    ▼ (requeue every 10s)
                               Check capacity
                                    │
                                    ├─── Capacity available ──► Phase: Running
                                    │
                                    └─── Still at capacity ──► Remain Queued
```

### Quota (Rate Limiting)

In addition to concurrent Task limits, Agents support rate limiting via `quota`:

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: rate-limited-agent
spec:
  profile: "Rate-limited agent for API-quota-constrained backends"
  agentImage: quay.io/kubeopencode/kubeopencode-agent-opencode:latest
  executorImage: quay.io/kubeopencode/kubeopencode-agent-devbox:latest
  workspaceDir: /workspace
  command: ["sh", "-c", "/tools/opencode run \"$(cat /workspace/task.md)\""]
  serviceAccountName: kubeopencode-agent
  quota:
    maxTaskStarts: 10     # Maximum 10 task starts
    windowSeconds: 3600   # Per hour (sliding window)
```

**Quota vs MaxConcurrentTasks:**

| Feature | `maxConcurrentTasks` | `quota` |
|---------|----------------------|---------|
| What it limits | Simultaneous running Tasks | Rate of new Task starts |
| Time component | No (instant check) | Yes (sliding window) |
| State tracking | None (counts Running Tasks) | `Agent.status.taskStartHistory` |
| Queued Reason | `AgentAtCapacity` | `QuotaExceeded` |
| Use case | Limit resource usage | API rate limiting |

Both can be used together for comprehensive control.

**Agent Status with Quota:**

When quota is configured, the Agent's status tracks recent Task starts:

```yaml
status:
  taskStartHistory:
    - taskName: "task-1"
      taskNamespace: "default"
      startTime: "2024-01-15T10:00:00Z"
    - taskName: "task-2"
      taskNamespace: "default"
      startTime: "2024-01-15T10:05:00Z"
```

Records are automatically pruned when they fall outside the sliding window.

### Persistence and Suspend

Agents support persistent storage and suspend/resume:

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: slack-agent
  namespace: platform-agents
spec:
  profile: "Persistent Slack bot agent for interactive sessions"
  executorImage: quay.io/kubeopencode/kubeopencode-agent-devbox:latest
  workspaceDir: /workspace
  serviceAccountName: kubeopencode-agent
  port: 4096                    # OpenCode server port (default: 4096)
  persistence:
    sessions:
      size: "2Gi"
    workspace:
      size: "10Gi"

  # Resource requirements
  podSpec:
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"

  maxConcurrentTasks: 10
```

**Agent Lifecycle:**

```
Agent Created
    │
    ▼
Agent Controller
    ├── Creates Deployment (opencode serve)
    └── Creates Service (ClusterIP)

Task Created (with agentRef)
    │
    ▼
Task Controller
    ├── Creates Pod with command: opencode run --attach <server-url> "task"
    ├── Standard Pod status tracking
    └── Logs available via kubectl logs

Task Created (with templateRef)
    │
    ▼
Task Controller
    ├── Creates standalone Pod with command: opencode run "task"
    ├── Uses template config (images, workspace, credentials)
    └── Pod terminates when done
```

**Persistence Fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `spec.port` | int32 | 4096 | Port for OpenCode server |
| `spec.persistence` | PersistenceConfig | nil | Persistent storage configuration |
| `spec.persistence.sessions` | VolumePersistence | nil | Session data (SQLite DB) persistence |
| `spec.persistence.workspace` | VolumePersistence | nil | Workspace directory persistence |
| `spec.suspend` | bool | false | Scale deployment to 0 replicas (suspend/resume) |

**VolumePersistence Fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `storageClassName` | *string | cluster default | StorageClass for the PVC |
| `size` | string | 1Gi | PVC storage size |

**Agent Status:**

```yaml
status:
  deploymentName: slack-agent-server
  serviceName: slack-agent
  url: http://slack-agent.platform-agents.svc.cluster.local:4096
  ready: true
  suspended: false
  conditions:
    - type: ServerReady
      status: "True"
      reason: DeploymentReady
    - type: ServerHealthy
      status: "True"
      reason: DeploymentHealthy
```

---

## System Configuration

### KubeOpenCodeConfig (System-level Configuration)

KubeOpenCodeConfig provides **cluster-wide** settings for container image configuration and Task cleanup policies.

> **Note**: KubeOpenCodeConfig is a **cluster-scoped singleton** resource. Following OpenShift convention, it must be named `cluster`.

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: KubeOpenCodeConfig
metadata:
  name: cluster  # Required singleton name
spec:
  # System image configuration for internal KubeOpenCode components
  # (git-init, context-init containers)
  systemImage:
    image: quay.io/kubeopencode/kubeopencode:latest  # Default system image
    imagePullPolicy: Always  # Always/Never/IfNotPresent (default: IfNotPresent)

  # Task cleanup configuration (optional)
  cleanup:
    # Delete completed/failed Tasks after 1 hour
    ttlSecondsAfterFinished: 3600
    # Keep at most 100 completed Tasks per namespace
    maxRetainedTasks: 100
```

**Field Description:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spec.systemImage.image` | string | No | System image for internal components (default: built-in DefaultKubeOpenCodeImage) |
| `spec.systemImage.imagePullPolicy` | string | No | Pull policy for system containers: Always, Never, IfNotPresent (default: IfNotPresent) |
| `spec.cleanup.ttlSecondsAfterFinished` | int32 | No | TTL in seconds for cleaning up finished Tasks. Tasks are deleted after this duration from CompletionTime. |
| `spec.cleanup.maxRetainedTasks` | int32 | No | Maximum number of completed/failed Tasks to retain per namespace. Oldest Tasks (by CompletionTime) are deleted first. |
| `spec.proxy` | *ProxyConfig | No | Cluster-wide HTTP/HTTPS proxy settings. Agent-level proxy overrides this. |
| `spec.proxy.httpProxy` | string | No | HTTP proxy URL |
| `spec.proxy.httpsProxy` | string | No | HTTPS proxy URL |
| `spec.proxy.noProxy` | string | No | Comma-separated bypass list (.svc,.cluster.local always appended) |

**Image Pull Policy:**

Setting `imagePullPolicy: Always` is recommended when:
- Using `:latest` tags in development/staging environments
- Nodes may have cached old images that differ from registry versions
- Frequent image updates are expected

The `systemImage` configuration affects all internal KubeOpenCode containers:
- `git-init`: Clones Git repositories for Context
- `context-init`: Copies ConfigMap content to writable workspace

**Task Cleanup:**

The cleanup configuration enables automatic garbage collection of completed/failed Tasks:

- **TTL-based cleanup**: Tasks are deleted after `ttlSecondsAfterFinished` seconds from completion
- **Retention-based cleanup**: Only the most recent `maxRetainedTasks` completed Tasks are retained per namespace
- **Combined**: Both policies can be used together. TTL is checked first, then retention count
- **Cascading deletion**: Deleting a Task automatically deletes its associated Pod and ConfigMap
- **Cluster-wide config, per-namespace retention**: The configuration is cluster-scoped, but `maxRetainedTasks` limit applies independently to each namespace

Cleanup is disabled by default. When `KubeOpenCodeConfig` is not present or `cleanup` is not specified, Tasks are never automatically deleted

---

## Complete Examples

### 1. Simple Task Execution (Agent)

```yaml
# Create Agent (creates Deployment + Service)
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: default
  namespace: kubeopencode-system
spec:
  profile: "Default development agent for general tasks"
  agentImage: quay.io/kubeopencode/kubeopencode-agent-opencode:latest
  executorImage: quay.io/kubeopencode/kubeopencode-agent-devbox:latest
  workspaceDir: /workspace
  serviceAccountName: kubeopencode-agent
---
# Create Task (runs on the Agent's server via --attach)
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: update-service-a
  namespace: kubeopencode-system
spec:
  agentRef:
    name: default
  description: |
    Update dependencies to latest versions.
    Run tests and create PR.
```

### 2. Task with Multiple Contexts

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: complex-task
  namespace: kubeopencode-system
spec:
  agentRef: my-agent
  description: "Refactor the authentication module"
  contexts:
    # ConfigMap context (specific key)
    - type: ConfigMap
      mountPath: /workspace/guide.md
      configMap:
        name: guides
        key: refactoring-guide.md
    # ConfigMap context (all keys as directory)
    - type: ConfigMap
      mountPath: /workspace/configs
      configMap:
        name: project-configs
    # Git context
    - type: Git
      mountPath: /workspace/repo
      git:
        repository: https://github.com/org/repo
        ref: main
```

### 3. Batch Operations with Helm

For running the same task across multiple targets, use Helm templating:

```yaml
# values.yaml
tasks:
  - name: update-service-a
    repo: service-a
  - name: update-service-b
    repo: service-b
  - name: update-service-c
    repo: service-c

# templates/tasks.yaml
{{- range .Values.tasks }}
---
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: {{ .name }}
spec:
  description: "Update dependencies for {{ .repo }}"
{{- end }}
```

```bash
# Generate and apply multiple tasks
helm template my-tasks ./chart | kubectl apply -f -
```

---

## kubectl Usage

### Task Operations

```bash
# Create a task
kubectl apply -f task.yaml

# List tasks
kubectl get tasks -n kubeopencode-system

# Watch task execution
kubectl get task update-service-a -n kubeopencode-system -w

# Check task status
kubectl get task update-service-a -o yaml

# View task logs
kubectl logs $(kubectl get task update-service-a -o jsonpath='{.status.podName}') -n kubeopencode-system

# Stop a running task (gracefully stops and marks as Completed with logs preserved)
kubectl annotate task update-service-a kubeopencode.io/stop=true

# Delete task
kubectl delete task update-service-a -n kubeopencode-system
```

### Agent Operations

```bash
# List agents
kubectl get agents -n kubeopencode-system

# Create agent
kubectl apply -f agent.yaml

# View agent details
kubectl get agent my-agent -o yaml
```

---

## Web UI

KubeOpenCode includes a web-based UI for managing Tasks and viewing Agents.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    kubeopencode server                       │
│  ┌─────────────────────┐    ┌─────────────────────────────┐ │
│  │     REST API        │    │    Embedded React UI        │ │
│  │  /api/v1/tasks      │    │    (TypeScript + React)     │ │
│  │  /api/v1/agents     │    │                             │ │
│  └──────────┬──────────┘    └─────────────────────────────┘ │
└─────────────┼───────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes API                            │
│         Tasks, Agents, Pods, Namespaces                      │
└─────────────────────────────────────────────────────────────┘
```

**Key Design:**
- Single server binary (`kubeopencode server` subcommand)
- React UI embedded in Go binary via `embed` package
- REST API with JSON responses
- ServiceAccount token authentication (Kubernetes RBAC)
- No external dependencies (no database)

### REST API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/namespaces/{ns}/tasks` | List Tasks |
| GET | `/api/v1/namespaces/{ns}/tasks/{name}` | Get Task |
| POST | `/api/v1/namespaces/{ns}/tasks` | Create Task |
| DELETE | `/api/v1/namespaces/{ns}/tasks/{name}` | Delete Task |
| POST | `/api/v1/namespaces/{ns}/tasks/{name}/stop` | Stop Task |
| GET | `/api/v1/namespaces/{ns}/tasks/{name}/logs` | Stream logs (SSE) |
| GET | `/api/v1/agents` | List all Agents |
| GET | `/api/v1/namespaces/{ns}/agents` | List Agents in namespace |
| GET | `/api/v1/namespaces/{ns}/agents/{name}` | Get Agent details |
| GET | `/api/v1/info` | Server info |
| GET | `/api/v1/namespaces` | List namespaces |

### UI Pages

| Page | Path | Description |
|------|------|-------------|
| Task List | `/tasks` | View and filter Tasks across namespaces |
| Task Detail | `/tasks/:namespace/:name` | Task details with real-time log streaming |
| Task Create | `/tasks/create` | Create new Tasks with Agent selection |
| Agent List | `/agents` | Browse available Agents |
| Agent Detail | `/agents/:namespace/:name` | View Agent configuration |

### Deployment

Enable the UI server in Helm:

```yaml
server:
  enabled: true
  replicas: 1
  service:
    type: ClusterIP
    port: 2746
```

Access via port-forward:

```bash
kubectl port-forward -n kubeopencode-system svc/kubeopencode-server 2746:2746
```

---

## Summary

**API**:
- **Task** - primary API for single task execution (supports both `agentRef` and `templateRef`)
- **Agent** - running AI agent instance (always creates Deployment + Service)
- **AgentTemplate** - reusable blueprint for Agents and ephemeral Tasks
- **KubeOpenCodeConfig** - system-level settings (systemImage, cleanup)

**Context Types** (via ContextItem):
- `Text` - Content directly in YAML
- `ConfigMap` - Content from ConfigMap (single key or all keys as directory)
- `Git` - Content from Git repository with branch/tag/commit support
- `Runtime` - KubeOpenCode platform awareness (environment variables, kubectl commands, system concepts)
- `URL` - Content fetched from remote HTTP/HTTPS URL at task execution time

**Namespace Model**:
- Task and Agent must be in the same namespace
- Pod runs in the same namespace as the Task and Agent
- OwnerReference-based cleanup for Pod and ConfigMap

**Task Lifecycle**:
- No retry on failure (AI tasks are non-idempotent)
- User-initiated stop via `kubeopencode.io/stop=true` annotation (graceful, Pod deleted)
- OwnerReference-based cleanup for Pod and ConfigMap

**Batch Operations**:
- Use Helm, Kustomize, or other templating tools
- Kubernetes-native approach

**Event-Driven Triggers**:
- Use [Argo Events](https://argoproj.github.io/argo-events/) for webhook-driven Task creation
- See the [kubeopencode/dogfooding](https://github.com/kubeopencode/dogfooding) repository for examples

**Workflow Orchestration**:
- Use [Argo Workflows](https://argoproj.github.io/argo-workflows/) for multi-stage task orchestration
- KubeOpenCode Tasks can be triggered from Argo Workflow steps

**Advantages**:
- Simplified Architecture
- Native Integration with K8s tools
- Declarative Management (GitOps ready)
- Infrastructure Reuse
- Simplified Operations

---

**Status**: FINAL
**Date**: 2026-01-17
**Version**: v0.1.0
**Maintainer**: KubeOpenCode Team
