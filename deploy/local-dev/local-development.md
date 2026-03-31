# Local Development Guide

This guide describes how to set up a local development environment for KubeOpenCode using Kind (Kubernetes in Docker).

## Prerequisites

- Docker
- Kind (`brew install kind` on macOS)
- kubectl
- Helm 3.x
- Go 1.25+

## Quick Start

### 1. Create or Use Existing Kind Cluster

Check if you already have a Kind cluster running:

```bash
kind get clusters
```

If you have an existing cluster (e.g., `kind`), you can use it directly. Otherwise, create a new one:

```bash
kind create cluster --name kubeopencode
```

Verify the cluster is running:

```bash
kubectl cluster-info
```

**Note:** The examples below use `--name kubeopencode` for Kind commands. If using an existing cluster with a different name (e.g., `kind`), replace `--name kubeopencode` with your cluster name.

### 2. Build Images

Build all required images:

```bash
# Build the controller image
make docker-build

# Build all agent images (opencode, devbox, attach, etc.)
make agent-build-all
```

Or build individual agent images as needed:

```bash
make agent-build AGENT=opencode    # OpenCode init container (copies /opencode binary)
make agent-build AGENT=devbox      # Executor container (development environment)
make agent-build AGENT=attach      # Attach image (required for Server mode)
```

> **Note:** The `attach` image is required for **Server mode** Agents. If you only use Pod mode, you can skip it. However, `make agent-build-all` is recommended to avoid missing images.

**Note:** The unified kubeopencode image provides both controller and infrastructure utilities:
- `kubeopencode controller`: Kubernetes controller
- `kubeopencode git-init`: Git repository cloning for Git Context
- `kubeopencode save-session`: Workspace persistence for session resume

### 3. Load Images to Kind

Load images into the Kind cluster (required because Kind cannot pull from local Docker):

```bash
# Load controller image
kind load docker-image quay.io/kubeopencode/kubeopencode:latest --name kubeopencode

# Load all agent images
for img in opencode devbox attach; do
  kind load docker-image quay.io/kubeopencode/kubeopencode-agent-${img}:latest --name kubeopencode
done
```

> **Important:** All three agent images must be loaded. Missing the `attach` image will cause Server mode Tasks to fail with `ErrImagePull`.

### 4. Deploy with Helm

```bash
helm upgrade --install kubeopencode ./charts/kubeopencode \
  --namespace kubeopencode-system \
  --create-namespace \
  --set controller.image.pullPolicy=Never \
  --set agent.image.pullPolicy=Never \
  --set server.enabled=true
```

> **Note:** The `server.enabled=true` deploys the UI server. You can omit it if you only need the controller.

### 5. Verify Deployment

Check the controller is running:

```bash
kubectl get pods -n kubeopencode-system
```

Expected output:

```
NAME                                       READY   STATUS    RESTARTS   AGE
kubeopencode-controller-xxxxxxxxx-xxxxx    1/1     Running   0          30s
kubeopencode-server-xxxxxxxxx-xxxxx        1/1     Running   0          30s
```

> **Note:** If `server.enabled=false`, only the controller pod will be present.

Check CRDs are installed:

```bash
kubectl get crds | grep kubeopencode
```

Expected output:

```
agents.kubeopencode.io               <timestamp>
agenttemplates.kubeopencode.io       <timestamp>
kubeopencodeconfigs.kubeopencode.io   <timestamp>
tasks.kubeopencode.io                <timestamp>
```

Check controller logs:

```bash
kubectl logs -n kubeopencode-system deployment/kubeopencode-controller
```

## UI Server

KubeOpenCode includes a web UI for managing Tasks and viewing Agents. The UI is an "Agent as Application" platform that allows non-technical users to interact with AI agents.

### Access the UI

#### Option 1: Port Forward (Quick Access)

```bash
kubectl port-forward -n kubeopencode-system svc/kubeopencode-server 2746:2746
```

Then open http://localhost:2746 in your browser.

#### Option 2: NodePort (Kind Cluster)

Update Helm values to expose via NodePort:

```bash
helm upgrade kubeopencode ./charts/kubeopencode \
  --namespace kubeopencode-system \
  --set server.enabled=true \
  --set server.service.type=NodePort \
  --set server.service.nodePort=32746
```

Access via: http://localhost:32746

#### Option 3: Ingress

Enable ingress in Helm values:

```bash
helm upgrade kubeopencode ./charts/kubeopencode \
  --namespace kubeopencode-system \
  --set server.enabled=true \
  --set server.ingress.enabled=true \
  --set server.ingress.hosts[0].host=kubeopencode.local \
  --set server.ingress.hosts[0].paths[0].path=/ \
  --set server.ingress.hosts[0].paths[0].pathType=Prefix
```

Add to `/etc/hosts`:
```
127.0.0.1 kubeopencode.local
```

### UI Features

| Feature | Description |
|---------|-------------|
| **Task List** | View all Tasks across namespaces with status filtering |
| **Task Detail** | View Task details, logs (real-time streaming) |
| **Task Create** | Create new Tasks with Agent selection (filtered by namespace permissions) |
| **Agent List** | Browse available Agents with namespace filter |
| **Agent Detail** | View Agent configuration, contexts, credentials |
| **AgentTemplate List** | Browse available AgentTemplates |
| **Filtering** | Filter resources by name and Kubernetes label selectors |
| **Pagination** | Server-side pagination for efficient browsing of large resource lists |

#### Resource Filtering

All list pages (Tasks, Agents) support filtering:

- **Name Filter**: Filter resources by name (substring match)
- **Label Selector**: Filter by Kubernetes labels using standard selector syntax (e.g., `app=myapp,env=prod`)

Filters are persisted in the URL as query parameters, making it easy to share filtered views with team members:

```
http://localhost:2746/agents?name=opencode&labels=env%3Dprod
```

#### Pagination

List pages use server-side pagination with 12 items per page. The pagination controls at the bottom of each list show:
- Current page range (e.g., "Showing 1 to 12 of 45")
- Previous/Next navigation buttons

#### Namespace Filtering

The Agent list page includes a namespace selector that allows you to filter Agents by namespace. This helps in multi-tenant environments where different teams have Agents in different namespaces.

#### Agent Availability

When creating a Task, only Agents in the same namespace as the Task are shown. Task and Agent must always be in the same namespace.

### Authentication

The UI uses ServiceAccount token authentication. For local development with port-forward, the server operates in a permissive mode suitable for testing.

For production, enable authentication with RBAC-based filtering:

```bash
helm upgrade kubeopencode ./charts/kubeopencode \
  --namespace kubeopencode-system \
  --set server.enabled=true \
  --set server.authEnabled=true \
  --set server.authAllowAnonymous=false
```

When authentication is enabled:
- Users must provide a Bearer token in the Authorization header
- The token is validated using the Kubernetes TokenReview API
- API requests are executed with user impersonation, respecting Kubernetes RBAC
- Users only see Agents and Tasks they have permission to access

### UI Development

To develop the UI locally with hot-reload:

```bash
# Terminal 1: Run the Go server (API backend)
make run-server

# Terminal 2: Run webpack dev server (frontend with hot-reload)
make ui-dev
```

The webpack dev server runs on http://localhost:3000 and proxies API requests to the Go server on port 2746.

To build the UI for production:

```bash
make ui-build
```

## Iterative Development

### Controller Changes

When you make changes to the controller code:

```bash
# Rebuild the image
make docker-build

# Reload into Kind
kind load docker-image quay.io/kubeopencode/kubeopencode:latest --name kubeopencode

# Restart the deployment to pick up the new image
kubectl rollout restart deployment/kubeopencode-controller -n kubeopencode-system

# Watch the rollout
kubectl rollout status deployment/kubeopencode-controller -n kubeopencode-system
```

Or use the convenience target:

```bash
make e2e-reload
```

### UI Changes

When you make changes to the UI code:

```bash
# Rebuild the UI and docker image
make ui-build
make docker-build

# Reload into Kind
kind load docker-image quay.io/kubeopencode/kubeopencode:latest --name kubeopencode

# Restart the server deployment
kubectl rollout restart deployment/kubeopencode-server -n kubeopencode-system
```

For faster iteration during UI development, use the dev server instead:

```bash
# Run Go server locally (uses kubeconfig for API access)
make run-server

# In another terminal, run webpack dev server with hot-reload
make ui-dev
```

This provides instant feedback without rebuilding Docker images.

## Local Test Environment

For quick testing, use the pre-configured resources in `deploy/local-dev/`:

### Deploy Test Resources

```bash
# First, create secrets.yaml from template
cp deploy/local-dev/secrets.yaml.example deploy/local-dev/secrets.yaml
# Edit secrets.yaml with your real API keys
vim deploy/local-dev/secrets.yaml

# Deploy all resources (namespace, secrets, RBAC, template, agents)
kubectl apply -k deploy/local-dev/

# Verify the AgentTemplate
kubectl get agenttemplate -n test

# Verify the Agents are ready
kubectl get agent -n test

# For Server mode, verify the deployment is created
kubectl get deployment -n test
```

### Resources Created

| Resource | Name | Description |
|----------|------|-------------|
| Namespace | `test` | Isolated namespace for testing |
| Secret | `opencode-credentials` | OpenCode API key |
| Secret | `git-settings` | Git author/committer settings |
| ServiceAccount | `kubeopencode-agent` | Agent service account |
| Role/RoleBinding | `kubeopencode-agent` | RBAC permissions |
| AgentTemplate | `local-dev-base` | Shared base configuration (images, credentials, workspace) |
| Agent | `server-agent` | Server-mode agent with session + workspace persistence |
| Agent | `pod-agent` | Pod-mode agent (ephemeral, per-task) |

### Features Demonstrated

The local-dev resources showcase the following features:

| Feature | Resource | Description |
|---------|----------|-------------|
| **AgentTemplate** | `local-dev-base` | Shared config inherited by both Agents via `templateRef` |
| **Server Mode** | `server-agent` | Persistent OpenCode server (Deployment + Service) |
| **Session Persistence** | `server-agent` | Conversation history survives pod restarts (1Gi PVC) |
| **Workspace Persistence** | `server-agent` | Git repos and files survive pod restarts (5Gi PVC) |
| **Suspend/Resume** | `server-agent` | Can be suspended to save compute (see below) |
| **Concurrency Control** | `server-agent` | Limited to 3 concurrent tasks |
| **Pod Mode** | `pod-agent` | Ephemeral one-Pod-per-Task execution |
| **Agent Profile** | Both agents | Human-readable description for discovery |

### Test Tasks

#### Server Mode Test

```bash
kubectl apply -n test -f - <<EOF
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: server-test
spec:
  agentRef:
    name: server-agent
  description: "Say hello world"
EOF

# Check status
kubectl get task -n test
kubectl logs -n test server-test-pod -c agent
```

#### Pod Mode Test

```bash
kubectl apply -n test -f - <<EOF
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: pod-test
spec:
  agentRef:
    name: pod-agent
  description: "What is 2+2?"
EOF

# Check status
kubectl get task -n test
kubectl logs -n test pod-test-pod -c agent
```

#### Concurrent Tasks Test

```bash
for i in 1 2 3; do
  kubectl apply -n test -f - <<EOF
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: concurrent-$i
spec:
  agentRef:
    name: server-agent
  description: "Count to $i"
EOF
done

# Watch progress
kubectl get task -n test -w
```

### Testing Persistence

Session and workspace persistence let the server-agent retain state across pod restarts.

#### Verify PVCs Are Created

```bash
# After deploying the server-agent, check for PVCs
kubectl get pvc -n test

# Expected output:
# NAME                             STATUS   VOLUME   CAPACITY   AGE
# server-agent-server-sessions     Bound    ...      1Gi        ...
# server-agent-server-workspace    Bound    ...      5Gi        ...
```

#### Test Session Persistence

```bash
# Run a task to create a conversation
kubectl apply -n test -f - <<EOF
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: persist-test-1
spec:
  agentRef:
    name: server-agent
  description: "Remember that the secret code is 42"
EOF

# Wait for completion, then restart the server pod
kubectl rollout restart deployment/server-agent-server -n test
kubectl rollout status deployment/server-agent-server -n test

# The session history should survive the restart
```

#### Test Workspace Persistence

```bash
# Run a task that creates files in the workspace
kubectl apply -n test -f - <<EOF
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: persist-test-2
spec:
  agentRef:
    name: server-agent
  description: "Create a file called hello.txt with the content 'Hello from KubeOpenCode'"
EOF

# Restart the server pod
kubectl rollout restart deployment/server-agent-server -n test

# The workspace files should still be there after restart
```

### Testing Suspend/Resume

Server-mode agents can be suspended to save compute resources while retaining all data.

#### Suspend the Agent

```bash
# Edit the agent to set suspend: true
kubectl patch agent server-agent -n test --type=merge -p '
spec:
  serverConfig:
    suspend: true
'

# Verify the deployment is scaled to 0
kubectl get deployment -n test
# Expected: server-agent-server   0/0

# Check agent status
kubectl get agent server-agent -n test -o jsonpath='{.status.serverStatus.suspended}'
# Expected: true

# PVCs are retained (no data loss)
kubectl get pvc -n test
```

#### Tasks Queue While Suspended

```bash
# Create a task while agent is suspended
kubectl apply -n test -f - <<EOF
apiVersion: kubeopencode.io/v1alpha1
kind: Task
metadata:
  name: queued-test
spec:
  agentRef:
    name: server-agent
  description: "This will queue until the agent is resumed"
EOF

# Check task status - should be Queued with reason AgentSuspended
kubectl get task queued-test -n test -o jsonpath='{.status.phase}'
# Expected: Queued
```

#### Resume the Agent

```bash
# Resume the agent
kubectl patch agent server-agent -n test --type=merge -p '
spec:
  serverConfig:
    suspend: false
'

# Verify the deployment scales back up
kubectl get deployment -n test
# Expected: server-agent-server   1/1

# Queued tasks should automatically start running
kubectl get task -n test -w
```

### Testing AgentTemplate

The `local-dev-base` AgentTemplate provides shared configuration for both agents.

#### View Template Configuration

```bash
# View the template
kubectl get agenttemplate local-dev-base -n test -o yaml

# Check which agents reference this template
kubectl get agent -n test -l kubeopencode.io/agent-template=local-dev-base
```

#### Verify Template Inheritance

```bash
# Both agents should have the label set by the controller
kubectl get agent -n test --show-labels
# Expected labels include: kubeopencode.io/agent-template=local-dev-base
```

### Customization

#### Using Real Secrets

Create a local secrets file (gitignored):

```bash
cp deploy/local-dev/secrets.yaml deploy/local-dev/secrets.local.yaml
# Edit secrets.local.yaml with real values
kubectl apply -f deploy/local-dev/secrets.local.yaml -n test
```

#### Different AI Model

Edit the `agenttemplate.yaml` to change the model for all agents at once:

```yaml
config: |
  {
    "$schema": "https://opencode.ai/config.json",
    "model": "anthropic/claude-sonnet-4-20250514",
    "small_model": "anthropic/claude-haiku-4-20250514"
  }
```

Or override in a specific agent's config (agent-level config overrides template config).

#### Adjusting Persistence Sizes

Edit `agent-server.yaml` to change PVC sizes:

```yaml
serverConfig:
  port: 4096
  persistence:
    sessions:
      size: "2Gi"              # Increase session storage
      storageClassName: "gp3"  # Use a specific StorageClass
    workspace:
      size: "20Gi"             # Increase workspace storage
```

> **Note:** On Kind clusters, the default StorageClass (`standard`) is used. PVC resizing may not be supported depending on the provisioner.

### Stopping a Running Task

```bash
# Stop a task gracefully (Pod is deleted, Task marked as Stopped)
kubectl annotate task server-test kubeopencode.io/stop=true -n test

# Check status
kubectl get task server-test -n test -o jsonpath='{.status.phase}'
# Expected: Completed (with Stopped condition)
```

> **Note:** Logs are lost when a Task is stopped. Use `kubectl logs` before stopping to capture output.

## Cleanup

### Delete Test Resources

```bash
# Delete all tasks
kubectl delete task --all -n test

# Delete all test resources (PVCs are cleaned up via OwnerReference)
kubectl delete -k deploy/local-dev/
```

### Uninstall KubeOpenCode

```bash
helm uninstall kubeopencode -n kubeopencode-system
kubectl delete namespace kubeopencode-system
```

### Delete Kind Cluster

```bash
kind delete cluster --name kubeopencode
```

## Debugging Tools

### Reading OpenCode Stream JSON Output

When running Tasks with `--format json`, the output is in stream-json format which can be hard to read. We provide a utility script to format the output:

```bash
# Read from kubectl logs
kubectl logs <pod-name> -n kubeopencode-system | ./hack/opencode-stream-reader.sh

# Read from a saved log file
cat task-output.log | ./hack/opencode-stream-reader.sh
```

The script requires `jq` and converts the JSON stream into human-readable output with colors and formatting.

## Troubleshooting

### Image Pull Errors

If you see `ErrImagePull` or `ImagePullBackOff`, ensure:

1. Images are loaded into Kind: `docker exec kubeopencode-control-plane crictl images | grep kubeopencode`
2. `imagePullPolicy` is set to `Never` in Helm values
3. **Server mode:** The `attach` image (`kubeopencode-agent-attach`) must be loaded. This image is used by Server mode Task Pods to connect to the OpenCode server. Build and load it with:
   ```bash
   make agent-build AGENT=attach
   kind load docker-image quay.io/kubeopencode/kubeopencode-agent-attach:latest --name kubeopencode
   ```

### Controller Not Starting

Check controller logs:

```bash
kubectl logs -n kubeopencode-system deployment/kubeopencode-controller
```

Check events:

```bash
kubectl get events -n kubeopencode-system --sort-by='.lastTimestamp'
```

### CRDs Not Found

Ensure CRDs are installed:

```bash
kubectl get crds | grep kubeopencode
```

If missing, reinstall with Helm or apply manually:

```bash
kubectl apply -f deploy/crds/
```

### PVC Issues (Persistence)

If PVCs are stuck in `Pending`:

```bash
# Check PVC status
kubectl describe pvc -n test

# On Kind, ensure default StorageClass exists
kubectl get storageclass
```

Kind clusters include a `standard` StorageClass by default. If missing, recreate the cluster.
