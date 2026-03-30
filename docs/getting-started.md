# Getting Started with KubeOpenCode

This guide covers installation, configuration, and basic usage of KubeOpenCode.

## Prerequisites

- Kubernetes 1.25+
- Helm 3.8+

## Installation

### Install from OCI Registry

```bash
# Create namespace
kubectl create namespace kubeopencode-system

# Install from OCI registry (with UI enabled)
helm install kubeopencode oci://quay.io/kubeopencode/helm-charts/kubeopencode \
  --namespace kubeopencode-system \
  --set server.enabled=true
```

### Install from Local Chart (Development)

```bash
# Create namespace
kubectl create namespace kubeopencode-system

# Install from local chart
helm install kubeopencode ./charts/kubeopencode \
  --namespace kubeopencode-system \
  --set server.enabled=true
```

## Access the Web UI

```bash
# Port forward to access the UI
kubectl port-forward -n kubeopencode-system svc/kubeopencode-server 2746:2746

# Open http://localhost:2746 in your browser
```

The Web UI provides:
- **Task List**: View and filter Tasks across namespaces
- **Task Detail**: Monitor Task execution with real-time log streaming
- **Task Creation**: Create new Tasks with Agent selection
- **Agent Browser**: View available Agents and their configurations

## Choose Your Mode

KubeOpenCode supports two execution modes. Choose the one that fits your use case:

| | **Server Mode (Live Agent)** | **Pod Mode (Batch Tasks)** |
|---|---|---|
| **What** | Persistent AI agent running as a service | Ephemeral Pod per Task |
| **Best for** | Interactive coding, team-shared agents, Slack bots | Batch operations, CI/CD pipelines, one-off tasks |
| **Cold start** | None (server always running) | Yes (container startup per Task) |
| **Context sharing** | Shared across Tasks via server | Isolated per Task |
| **Interaction** | Web Terminal, CLI attach, API | Logs only |

## Server Mode: Live Agent (Recommended for Getting Started)

Server Mode deploys a persistent AI agent as a Kubernetes service. Your team can interact with it anytime — through the web terminal, CLI, or by submitting Tasks programmatically.

### 1. Create a Server-Mode Agent

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: dev-agent
  namespace: kubeopencode-system
spec:
  profile: "Interactive development agent"
  executorImage: quay.io/kubeopencode/kubeopencode-agent-devbox:latest
  workspaceDir: /workspace
  serviceAccountName: kubeopencode-agent

  # Enable Server Mode — creates a persistent Deployment + Service
  serverConfig:
    port: 4096
    persistence:
      sessions:
        size: "2Gi"   # Persist conversation history across restarts

  credentials:
    - name: api-key
      secretRef:
        name: ai-credentials
        key: api-key
      env: OPENCODE_API_KEY

  # Optional: pre-load your codebase
  contexts:
    - name: source-code
      type: Git
      git:
        repository: https://github.com/your-org/your-repo.git
        ref: main
      mountPath: code
```

### 2. Wait for the Agent to Be Ready

```bash
# Watch the Agent status
kubectl get agents -n kubeopencode-system -w

# NAME        PROFILE                         MODE     STATUS
# dev-agent   Interactive development agent    Server   Ready

# Check the created resources
kubectl get deploy,svc -n kubeopencode-system -l kubeopencode.io/agent=dev-agent
```

The controller automatically creates:
- A **Deployment** running the OpenCode server
- A **Service** for internal cluster access

### 3. Interact with the Live Agent

**Option A: CLI (recommended)**

```bash
# Install the CLI
go install github.com/kubeopencode/kubeopencode/cmd/kubeoc@latest

# Attach to the agent — opens an interactive OpenCode terminal
kubeoc agent attach dev-agent -n kubeopencode-system
```

**Option B: Web Terminal**

```bash
# Port forward to the KubeOpenCode dashboard
kubectl port-forward -n kubeopencode-system svc/kubeopencode-server 2746:2746

# Open http://localhost:2746, navigate to the agent, and click "Terminal"
```

**Option C: Submit Tasks programmatically**

Even in Server Mode, you can submit Tasks. They run on the persistent server instead of creating new Pods:

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: fix-bug-123
  namespace: kubeopencode-system
spec:
  agentRef:
    name: dev-agent
  description: |
    Fix the null pointer exception in UserService.java.
    The bug is reported in issue #123.
```

### 4. Monitor and Manage

```bash
# View agent server logs
kubectl logs -n kubeopencode-system deploy/dev-agent-server -f

# Check server health
kubectl get agent dev-agent -n kubeopencode-system -o jsonpath='{.status.serverStatus}'

# Stop the agent (scales down the Deployment)
kubectl delete agent dev-agent -n kubeopencode-system
```

## Pod Mode: Batch Tasks

Pod Mode creates an ephemeral Pod for each Task — ideal for batch operations, CI/CD pipelines, and one-off tasks.

### 1. Create an Agent

KubeOpenCode uses a **two-container pattern**:
- **Init Container** (`agentImage`): Copies OpenCode binary to `/tools` shared volume
- **Worker Container** (`executorImage`): Runs tasks using `/tools/opencode`

```yaml
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
  credentials:
    - name: opencode-api-key
      secretRef:
        name: ai-credentials
        key: opencode-key
      env: OPENCODE_API_KEY
```

### 2. Create a Task

```yaml
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: update-service-a
  namespace: kubeopencode-system
spec:
  # Task description (becomes /workspace/task.md)
  description: |
    Update dependencies to latest versions.
    Run tests and create PR.

  # Optional inline contexts
  contexts:
    - type: Text
      text: |
        # Coding Standards
        - Use descriptive names
        - Write unit tests
```

### 3. Monitor Progress

```bash
# Watch Task status
kubectl get tasks -n kubeopencode-system -w

# Check detailed status
kubectl describe task update-service-a -n kubeopencode-system

# View task logs
kubectl logs $(kubectl get task update-service-a -o jsonpath='{.status.podName}') -n kubeopencode-system
```

## Batch Operations with Helm

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

## Next Steps

- [Features](features.md) - Learn about the context system, concurrency control, and more
- [Agent Images](agent-images.md) - Build custom agent images
- [Security](security.md) - RBAC, credential management, and best practices
- [Architecture](architecture.md) - System design and API reference
