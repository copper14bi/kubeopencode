# ADR 0019: Web Terminal Credential Security Strategy

**Status:** Accepted
**Date:** 2026-03-27
**Related:** [ADR 0018](0018-web-terminal-replaces-web-ui.md)

## Context

KubeOpenCode's web terminal (ADR 0018) provides browser-based shell access to agent server pods via `opencode attach`. These pods may contain credentials (API keys, tokens, SSH keys) mounted as environment variables or files through the Agent's `credentials` spec.

**Security concern:** A user with terminal access can read any credential visible to the shell process — `echo $API_KEY`, `cat /run/secrets/token`, or `cat /proc/self/environ`. This is not a bug in KubeOpenCode; it is a fundamental property of how operating systems work.

**Industry survey:** No Kubernetes web terminal product prevents in-shell credential reading:

| Tool | Hides credentials in shell? | Security model |
|------|:--:|---|
| OpenShift Web Terminal | No | Per-user RBAC + DevWorkspace isolation |
| Rancher | No | RBAC only |
| Kubernetes Dashboard / Headlamp | No | RBAC only |
| Lens | No | Local kubeconfig + RBAC |

Red Hat's current direction (External Secrets Operator GA, VaultDynamicSecret) is to eliminate static secrets entirely rather than attempt to hide them.

## Decision

### Short-Term: RBAC Enforcement and Session Controls (Implemented)

1. **Impersonation-based RBAC**: The terminal handler impersonates the authenticated user when calling `pods/exec`, so the Kubernetes API server enforces the user's own `pods/exec` permission — not the controller's service account. Log streaming also uses impersonated clientset for `pods/log` access.

2. **Same-origin WebSocket check**: Prevents cross-site WebSocket hijacking by validating the `Origin` header.

3. **Session idle timeout**: Terminal WebSocket connections are automatically closed after 30 minutes of inactivity (no user input). The timeout resets on each keystroke.

4. **Token validation**: All API requests are authenticated via Kubernetes TokenReview before reaching any handler.

5. **Auth header stripping**: The server removes `Authorization` headers before proxying to internal agent services, preventing token leakage to agent pods.

### Medium-Term: Namespace-Scoped RBAC for Teams

Administrators grant web UI access per namespace using a ClusterRole + RoleBinding pattern. Required permissions for web terminal and log viewing:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubeopencode-web-user
rules:
  # View tasks and agents
  - apiGroups: ["kubeopencode.io"]
    resources: ["tasks", "agents"]
    verbs: ["get", "list", "watch"]
  # Create and manage tasks
  - apiGroups: ["kubeopencode.io"]
    resources: ["tasks"]
    verbs: ["create", "delete", "patch"]
  # View pod status
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
  # Stream task logs (requires pods/log)
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
  # Web terminal access (requires pods/exec)
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]
```

Bind per namespace:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: team-a-kubeopencode-user
  namespace: team-a
subjects:
  - kind: Group
    name: team-a-devs
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: kubeopencode-web-user
  apiGroup: rbac.authorization.k8s.io
```

### Long-Term: Accept Exposure, Minimize Blast Radius

#### Why Credential Hiding Is Impossible

A fundamental constraint exists: **if a process needs a credential to function, the process owner can read it.** This is not a KubeOpenCode limitation — it is an operating system property that no Kubernetes product has solved.

Consider the AI agent's toolchain:
- `gh pr list` → checks `GITHUB_TOKEN` env var **before** making any HTTP request
- OpenCode → reads `OPENAI_API_KEY` from env at startup
- `git push` → reads credentials from `~/.git-credentials` or credential helper

These tools require credentials in their process environment (env vars or files). A sidecar proxy that injects HTTP headers cannot work because the tools **check for credential existence before sending any request** — if the env var is missing, they error out immediately without ever reaching the network layer.

**Industry consensus:** No production Kubernetes tool attempts to hide credentials from shell users. Red Hat's OpenShift, Rancher, Dashboard, Lens — all accept this reality. The industry has converged on a different strategy: **make credential exposure tolerable** by limiting blast radius.

#### Strategy: Short-Lived Credentials + Audit Trail

Since credentials must be readable by the agent process (and therefore by the terminal user), the practical approach is:

1. **Short-lived credentials**: Credentials expire quickly, so even if read via terminal, they are useless shortly after.
2. **Audit logging**: Record who accessed what terminal and when, enabling incident response.
3. **Least-privilege RBAC**: Only users who need terminal access get `pods/exec` permission.

#### Approach 1: Vault Dynamic Secrets (Recommended)

A sidecar continuously refreshes short-lived credentials into files that the worker container reads:

```
┌──────────────────────────────────────────────────────┐
│ Agent Server Pod                                      │
│                                                       │
│ ┌──────────────────────┐  ┌────────────────────────┐ │
│ │  Worker Container     │  │  Vault Agent Sidecar   │ │
│ │  (opencode + terminal)│  │                        │ │
│ │                       │  │  - Authenticates to    │ │
│ │  Reads credentials    │  │    Vault via K8s SA    │ │
│ │  from shared volume:  │  │  - Writes short-lived  │ │
│ │  /vault/secrets/      │◄─┤    tokens to shared    │ │
│ │                       │  │    volume every 5 min  │ │
│ │  GITHUB_TOKEN=...     │  │  - Old tokens auto-    │ │
│ │  (expires in 10 min)  │  │    revoke on refresh   │ │
│ └──────────────────────┘  └────────────────────────┘ │
│           │                                           │
│    shared emptyDir volume: /vault/secrets/             │
└──────────────────────────────────────────────────────┘
```

**How it works:**

1. Vault Agent sidecar authenticates to Vault using the pod's Kubernetes ServiceAccount
2. Vault generates a **short-lived** GitHub token (10-minute TTL) and writes it to `/vault/secrets/github-token`
3. Worker container reads `GITHUB_TOKEN` from that file (via env `GITHUB_TOKEN=$(cat /vault/secrets/github-token)` or credential helper)
4. Every 5 minutes, Vault Agent requests a new token and atomically replaces the file
5. Old tokens are automatically revoked by Vault

**What happens if a terminal user reads the token?**
- They get a token that expires in at most 10 minutes
- Vault audit log records every token issuance
- The blast radius is bounded by TTL, not by detection speed

**What changes for the agent?** Nothing. Credentials appear as normal env vars or files. Tools like `gh`, `git`, and OpenCode work exactly as before — they read from the same paths they always read from.

#### Approach 2: Workload Identity Federation (Cloud-Native)

For cloud-hosted clusters (GKE, EKS, AKS), use platform-native workload identity:

- **GKE Workload Identity**: Pod's ServiceAccount token is exchanged for a GCP access token via metadata server
- **AWS IRSA**: Pod assumes an IAM role, SDK automatically refreshes credentials via OIDC
- **Azure Workload Identity**: Federated identity credential, token exchange via projected volume

These work transparently because cloud SDKs already know how to use them. The credentials are short-lived (typically 1 hour) and auto-rotated.

#### Approach 3: Audit Logging (Defense-in-Depth)

Regardless of credential strategy, log all terminal access:

```
terminal session started  user=alice  agent=opencode-agent  namespace=team-a  pod=opencode-agent-server-abc123
terminal session ended    user=alice  duration=12m34s  idle_timeout=false
```

This does not prevent credential reading, but enables:
- **Incident response**: Who had shell access to which pod at what time?
- **Compliance**: Auditable record of all interactive sessions
- **Deterrence**: Users know their sessions are logged

#### What We Explicitly Do NOT Pursue

| Approach | Why rejected |
|----------|-------------|
| **Sidecar HTTP proxy** | Tools check credential existence before making HTTP calls; proxy never gets invoked |
| **Restricted shells (rbash)** | Trivially bypassed via Python/Perl one-liners or `/proc/self/environ` |
| **Removing `cat`/`echo` from images** | Any language runtime can read files; breaks agent functionality |
| **AppArmor/SELinux deny rules on secret paths** | Blocks the agent process too — it needs those same paths to function |
| **Kernel keyring** | Only works for TLS keys consumed by kernel; not applicable to API tokens |

## Consequences

### Positive

- **RBAC enforcement is sound**: Users cannot access terminals or logs they lack Kubernetes permissions for.
- **Idle timeout reduces exposure**: Forgotten tabs don't hold exec connections indefinitely.
- **Transparent to AI agents**: Short-lived credential rotation does not change how tools read credentials — env vars and files work the same way.
- **Aligned with industry direction**: Same approach as OpenShift (External Secrets Operator), cloud-native workload identity, and Vault-based credential management.

### Negative

- **Credentials are readable from terminal**: This is an accepted trade-off. No Kubernetes web terminal product has solved this, and our analysis shows it is fundamentally impossible without breaking agent functionality.
- **Short-lived credentials add infrastructure**: Vault or Workload Identity requires additional cluster components.
- **Audit logging requires storage**: Terminal session logs need to be collected and retained.

### Risks

- **Controller impersonation privilege**: The server's `impersonate` permission, if compromised, could impersonate any user. Mitigated by: running the server with minimal privileges, network policies restricting access.
- **Token leakage window**: Even with short TTLs, there is a window during which a leaked token is valid. Mitigated by: keeping TTLs as short as practical (5-10 minutes), Vault token revocation on rotation.

## Implementation Phases

| Phase | Scope | Status |
|-------|-------|--------|
| 1. RBAC + session controls | Impersonation, idle timeout, auth header stripping | Done |
| 2. RBAC documentation | ClusterRole templates, per-namespace binding examples | Done |
| 3. Audit logging | Log terminal session lifecycle with user identity | TODO |
| 4. Short-lived credentials | Vault Agent sidecar integration, dynamic secret rotation | TODO |
| 5. Workload Identity | Cloud-native identity federation (GKE, EKS, AKS) | TODO |
