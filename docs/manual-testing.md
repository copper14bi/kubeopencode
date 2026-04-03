# Manual Testing Guide

This document records manual test cases that cannot be fully covered by automated tests. They require a running Kubernetes cluster (local-dev Kind), real user connections, or external dependencies like Git repositories.

Run `make local-dev-reload` to deploy the latest code before testing.

---

## Connection-Aware Standby (ADR 0028)

These tests verify that standby auto-suspend is deferred while users have active connections.

### Test 1: CLI Attach Prevents Auto-Suspend

**Precondition**: Agent with `standby.idleTimeout` configured, no active Tasks.

```bash
# 1. Create Agent
kubectl apply -f - <<EOF
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: test-heartbeat
  namespace: default
spec:
  executorImage: "quay.io/kubeopencode/kubeopencode-agent-devbox:latest"
  workspaceDir: /workspace
  port: 4096
  standby:
    idleTimeout: "3m"
EOF

# 2. Wait for Agent ready
kubectl wait agent test-heartbeat --for=jsonpath='{.status.ready}'=true --timeout=120s

# 3. Attach via CLI (keep open)
KUBEOPENCODE_KUBECONFIG=/tmp/kind-kubeopencode-kubeconfig.yaml \
  go run ./cmd/kubeoc/ agent attach test-heartbeat -n default

# 4. In another terminal, monitor:
watch -n10 'kubectl get agent test-heartbeat -o jsonpath="heartbeat={.metadata.annotations.kubeopencode\.io/last-connection-active} suspend={.spec.suspend} idleSince={.status.idleSince}" && echo'
```

**Verify (while attached)**:
- `heartbeat` annotation updates every ~60 seconds
- `suspend` stays `false` even after `idleTimeout` expires
- `idleSince` stays empty

**Verify (after Ctrl+C)**:
- Heartbeat stops updating
- ~2 minutes later: `idleSince` is set (staleness expired, idle timer starts)
- ~3 minutes after that: `suspend` becomes `true` (auto-suspend)

### Test 2: Web Terminal Prevents Auto-Suspend

**Precondition**: Same Agent as Test 1, kubeopencode-server accessible.

```bash
# 1. Port-forward to server
kubectl port-forward -n kubeopencode-system svc/kubeopencode-server 2746:2746

# 2. Open browser: http://localhost:2746
#    Navigate to Agent "test-heartbeat" → Terminal tab
```

**Verify (while terminal is open)**:
- `heartbeat` annotation updates every ~60 seconds
- Agent is not suspended despite idle timeout

**Verify (after closing terminal tab)**:
- Heartbeat stops updating immediately
- ~2 minutes later: `idleSince` is set
- ~3 minutes after that: `suspend` becomes `true`

### Test 3: CLI Heartbeat Warning on Missing RBAC

**Precondition**: A ServiceAccount with `get` but NOT `patch` permission on Agents.

```bash
# 1. Create limited ServiceAccount
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-limited-user
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-limited-access
  namespace: default
rules:
- apiGroups: ["kubeopencode.io"]
  resources: ["agents"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-limited-binding
  namespace: default
subjects:
- kind: ServiceAccount
  name: test-limited-user
  namespace: default
roleRef:
  kind: Role
  name: test-limited-access
  apiGroup: rbac.authorization.k8s.io
EOF

# 2. Generate kubeconfig for the SA
TOKEN=$(kubectl create token test-limited-user -n default --duration=1h)
CLUSTER_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
CLUSTER_CA=$(kubectl config view --minify --flatten -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')

cat > /tmp/test-limited-kubeconfig.yaml <<KUBEEOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: ${CLUSTER_CA}
    server: ${CLUSTER_SERVER}
  name: kind-kubeopencode
contexts:
- context:
    cluster: kind-kubeopencode
    user: test-limited-user
    namespace: default
  name: limited
current-context: limited
users:
- name: test-limited-user
  user:
    token: ${TOKEN}
KUBEEOF

# 3. Attach with limited kubeconfig
KUBEOPENCODE_KUBECONFIG=/tmp/test-limited-kubeconfig.yaml \
  go run ./cmd/kubeoc/ agent attach test-heartbeat -n default 2>&1
```

**Verify**: stderr outputs `Warning: connection heartbeat failed: ...Forbidden...`

### Test 4: StandbyConfigWarning on Short Idle Timeout

```bash
# Create Agent with idleTimeout < 2 minutes
kubectl apply -f - <<EOF
apiVersion: kubeopencode.io/v1alpha1
kind: Agent
metadata:
  name: test-short-timeout
  namespace: default
spec:
  executorImage: "quay.io/kubeopencode/kubeopencode-agent-devbox:latest"
  workspaceDir: /workspace
  port: 4096
  standby:
    idleTimeout: "30s"
EOF

# Wait a few seconds, then check conditions
sleep 10
kubectl get agent test-short-timeout -o jsonpath='{.status.conditions}' | jq '.[] | select(.type == "StandbyConfigWarning")'
```

**Verify**: `StandbyConfigWarning` condition is `True` with reason `IdleTimeoutTooShort`.

---

## Git Auto-Sync (ADR 0027)

These tests require a real Git repository that you can push to.

### Test 5: Rollout on Remote Change

**Precondition**: Agent with `sync.policy: Rollout` pointing to a repo you can push to.

1. Create Agent with Rollout sync (interval ~30-60s)
2. Wait for initial `status.gitSyncStatuses[].commitHash` to be set
3. Push a new commit to the remote repo
4. Wait for one sync interval
5. **Verify**: commit hash in status and pod template annotation updates to the new hash
6. **Verify**: old Pod enters `Terminating`, new Pod starts `Running`

### Test 6: Task Protection Delays Rollout

**Precondition**: Agent with `sync.policy: Rollout` and a way to create a long-running Task.

1. Create Agent with Rollout sync
2. Create a Task against the Agent (Task stays in Running state)
3. Push a new commit to the remote repo
4. Wait for one sync interval
5. **Verify**: `GitSyncPending` condition becomes `True` with message "Waiting for N active task(s)..."
6. **Verify**: Pod is NOT restarted (annotation keeps old hash)
7. Stop the Task (`kubectl annotate task <name> kubeopencode.io/stop=true`)
8. **Verify**: within seconds, `GitSyncPending` becomes `False`, hash updates, rollout executes

---

## Cleanup

```bash
kubectl delete agent test-heartbeat test-short-timeout --ignore-not-found
kubectl delete rolebinding test-limited-binding --ignore-not-found
kubectl delete role test-limited-access --ignore-not-found
kubectl delete sa test-limited-user --ignore-not-found
rm -f /tmp/test-limited-kubeconfig.yaml
```
