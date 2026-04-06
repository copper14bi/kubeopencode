# ADR 0030: Graceful Task Termination on Deletion

## Status

Proposed

## Date

2026-04-06

## Context

When a user deletes a Task resource (`kubectl delete task xxx`), the running agent may still complete its work instead of stopping immediately. This is because the current implementation has no active termination mechanism on deletion.

### Current Behavior

1. User runs `kubectl delete task xxx`
2. Task reconciler sees `NotFound` and returns immediately (no-op)
3. The task Pod is deleted asynchronously by Kubernetes garbage collector via OwnerReference (background propagation by default)
4. Pod receives SIGTERM with default 30-second graceful termination period
5. The OpenCode process inside the Pod does not handle SIGTERM properly, so it continues running until completion or forced kill (SIGKILL after 30s)

### Result

The agent may finish its entire task even after the user explicitly deleted the Task, which violates user expectations.

### Comparison with Stop Annotation

The `kubeopencode.io/stop=true` annotation works slightly better because the controller explicitly deletes the Pod in `handleStop()`. However, it still relies on SIGTERM handling and does not actively signal the OpenCode process to abort.

## Proposed Improvements

### Phase 1: Finalizer-Based Deletion (Recommended First Step)

Add a finalizer to Task resources so the controller can intercept deletion and actively clean up:

1. Add finalizer `kubeopencode.io/task-cleanup` during task initialization
2. On deletion (finalizer triggered):
   - Explicitly delete the Pod (instead of relying on async GC)
   - Wait for Pod termination
   - Remove the finalizer to allow Task deletion to complete

### Phase 2: Shorter Graceful Termination Period

Set `terminationGracePeriodSeconds` to a shorter value (e.g., 5-10 seconds) on task Pods, since there is no meaningful cleanup work to do — the agent should stop as quickly as possible.

### Phase 3: SIGTERM Handling in OpenCode

Ensure the OpenCode process properly handles SIGTERM signals and exits promptly when received. This is an upstream OpenCode concern.

### Phase 4: Active Session Cancellation (agentRef Only)

For tasks using `agentRef` (connecting to a persistent Agent server), the controller should call the Agent server's API to cancel the running session before deleting the client Pod. Without this, the Agent server will continue executing the task even after the client Pod is gone.

## Decision

Record this as a known limitation. The recommended approach is to implement Phase 1 and Phase 2 first, as they are straightforward changes within the KubeOpenCode codebase. Phase 3 depends on upstream OpenCode changes. Phase 4 is only needed for `agentRef` tasks.

In the meantime, users should prefer `kubectl annotate task <name> kubeopencode.io/stop=true` over `kubectl delete task` to stop running tasks.

## Consequences

- **Positive**: Users will have predictable task termination behavior
- **Positive**: Reduced resource waste from tasks that continue running after deletion
- **Negative**: Finalizers add slight complexity to the deletion flow and could block deletion if the controller is down
