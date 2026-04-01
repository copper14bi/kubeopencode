// Copyright Contributors to the KubeOpenCode project

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubeopenv1alpha1 "github.com/kubeopencode/kubeopencode/api/v1alpha1"
)

const (
	// AgentConditionServerReady indicates whether the OpenCode server is ready.
	AgentConditionServerReady = "ServerReady"

	// AgentConditionServerHealthy indicates whether the Agent is responding to health checks.
	// Based on Deployment readiness rather than HTTP health checks.
	AgentConditionServerHealthy = "ServerHealthy"

	// AgentConditionSuspended indicates whether the Agent is intentionally suspended.
	AgentConditionSuspended = "Suspended"

	// DefaultServerReconcileInterval is how often to reconcile Agents.
	DefaultServerReconcileInterval = 30 * time.Second
)

// AgentReconciler reconciles Agent resources.
// It manages the Deployment and Service for each Agent.
type AgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubeopencode.io,resources=agents,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=kubeopencode.io,resources=agents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeopencode.io,resources=tasks,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;delete

// Reconcile handles Agent reconciliation.
// It ensures the Deployment and Service exist and are up-to-date.
func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Agent
	var agent kubeopenv1alpha1.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			// Agent was deleted, nothing to do (Deployment/Service will be garbage collected)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Agent")
		return ctrl.Result{}, err
	}

	// Manage agent-template label
	if err := r.reconcileTemplateLabel(ctx, &agent); err != nil {
		logger.Error(err, "Failed to reconcile template label")
		return ctrl.Result{}, err
	}

	// Resolve agent configuration (merge with template if referenced).
	agentCfg, err := r.resolveAgentConfig(ctx, &agent)
	if err != nil {
		logger.Error(err, "Failed to resolve agent config")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling Agent", "agent", agent.Name)
	sysCfg := r.getSystemConfig(ctx)

	// Apply cluster-level defaults where Agent doesn't specify its own
	agentCfg.applySystemDefaults(sysCfg)

	// Process Agent contexts (Text, ConfigMap, Git, Runtime)
	contextConfigMap, fileMounts, dirMounts, gitMounts, err := r.processAgentContexts(ctx, &agent, agentCfg)
	if err != nil {
		logger.Error(err, "Failed to process Agent contexts")
		return ctrl.Result{}, err
	}

	// Reconcile context ConfigMap if there are any contexts to store
	if err := r.reconcileContextConfigMap(ctx, &agent, contextConfigMap); err != nil {
		logger.Error(err, "Failed to reconcile context ConfigMap")
		return ctrl.Result{}, err
	}

	// Handle idle timeout (auto-suspend/auto-resume).
	// Only mutates in-memory status — updateAgentStatus persists all changes atomically.
	if agent.Spec.IdleTimeout != nil && !agent.Spec.Suspend {
		if err := r.reconcileIdleTimeout(ctx, &agent); err != nil {
			logger.Error(err, "Failed to reconcile idle timeout")
			return ctrl.Result{}, err
		}
	} else if agent.Spec.IdleTimeout == nil && agent.Status.IdleSince != nil {
		// idleTimeout was removed — clear idle tracking (persisted by updateAgentStatus)
		agent.Status.IdleSince = nil
	}

	// Evaluate auto-suspend once for consistency across reconcileDeployment and updateAgentStatus
	autoSuspended := r.shouldAutoSuspend(&agent)

	// Reconcile persistence PVCs if configured
	if err := r.reconcilePVC(ctx, &agent, BuildServerSessionPVC, "session"); err != nil {
		logger.Error(err, "Failed to reconcile session PVC")
		return ctrl.Result{}, err
	}
	if err := r.reconcilePVC(ctx, &agent, BuildServerWorkspacePVC, "workspace"); err != nil {
		logger.Error(err, "Failed to reconcile workspace PVC")
		return ctrl.Result{}, err
	}

	// Reconcile the Deployment (with context support)
	if err := r.reconcileDeployment(ctx, &agent, autoSuspended, agentCfg, sysCfg, contextConfigMap, fileMounts, dirMounts, gitMounts); err != nil {
		logger.Error(err, "Failed to reconcile Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile the Service
	if err := r.reconcileService(ctx, &agent); err != nil {
		logger.Error(err, "Failed to reconcile Service")
		return ctrl.Result{}, err
	}

	// Update Agent status
	if err := r.updateAgentStatus(ctx, &agent, autoSuspended); err != nil {
		logger.Error(err, "Failed to update Agent status")
		return ctrl.Result{}, err
	}

	// Calculate optimal requeue interval.
	// When idle timer is running, requeue precisely when timeout expires.
	requeueAfter := DefaultServerReconcileInterval
	if agent.Spec.IdleTimeout != nil && !agent.Spec.Suspend && agent.Status.IdleSince != nil && !autoSuspended {
		remaining := time.Until(agent.Status.IdleSince.Time.Add(agent.Spec.IdleTimeout.Duration))
		if remaining > 0 && remaining < requeueAfter {
			requeueAfter = remaining
		}
	}
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// reconcileDeployment ensures the Deployment exists and is up-to-date.
func (r *AgentReconciler) reconcileDeployment(ctx context.Context, agent *kubeopenv1alpha1.Agent, autoSuspended bool, agentCfg agentConfig, sysCfg systemConfig, contextConfigMap *corev1.ConfigMap, fileMounts []fileMount, dirMounts []dirMount, gitMounts []gitMount) error {
	logger := log.FromContext(ctx)

	desired := BuildServerDeployment(agent, agentCfg, sysCfg, contextConfigMap, fileMounts, dirMounts, gitMounts)

	// Scale to 0 replicas when manually suspended or auto-suspended due to idle timeout
	if agent.Spec.Suspend || autoSuspended {
		replicas := int32(0)
		desired.Spec.Replicas = &replicas
	}

	// Set owner reference for garbage collection
	if err := controllerutil.SetControllerReference(agent, desired, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Check if Deployment exists
	var existing appsv1.Deployment
	err := r.Get(ctx, client.ObjectKey{Namespace: desired.Namespace, Name: desired.Name}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create the Deployment
			logger.Info("Creating Deployment for Agent", "deployment", desired.Name)
			if err := r.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create Deployment: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get Deployment: %w", err)
	}

	// Update the Deployment if needed
	// For now, we do a simple update of the spec
	existing.Spec = desired.Spec
	existing.Labels = desired.Labels
	if err := r.Update(ctx, &existing); err != nil {
		return fmt.Errorf("failed to update Deployment: %w", err)
	}

	return nil
}

// reconcileService ensures the Service exists and is up-to-date.
func (r *AgentReconciler) reconcileService(ctx context.Context, agent *kubeopenv1alpha1.Agent) error {
	logger := log.FromContext(ctx)

	desired := BuildServerService(agent)

	// Set owner reference for garbage collection
	if err := controllerutil.SetControllerReference(agent, desired, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Check if Service exists
	var existing corev1.Service
	err := r.Get(ctx, client.ObjectKey{Namespace: desired.Namespace, Name: desired.Name}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Create the Service
			logger.Info("Creating Service for Agent", "service", desired.Name)
			if err := r.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create Service: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get Service: %w", err)
	}

	// Update the Service if needed
	// Preserve ClusterIP as it's immutable
	desired.Spec.ClusterIP = existing.Spec.ClusterIP
	existing.Spec = desired.Spec
	existing.Labels = desired.Labels
	if err := r.Update(ctx, &existing); err != nil {
		return fmt.Errorf("failed to update Service: %w", err)
	}

	return nil
}

// updateAgentStatus updates the Agent's status with deployment information.
// Health is determined by Deployment readiness (liveness/readiness probes on the Deployment
// already check the server's /session/status endpoint).
func (r *AgentReconciler) updateAgentStatus(ctx context.Context, agent *kubeopenv1alpha1.Agent, autoSuspended bool) error {
	deploymentName := ServerDeploymentName(agent.Name)
	agent.Status.DeploymentName = deploymentName
	agent.Status.ServiceName = ServerServiceName(agent.Name)
	agent.Status.URL = ServerURL(agent.Name, agent.Namespace, GetServerPort(agent))

	// Handle suspended state (manual or auto-suspend)
	if agent.Spec.Suspend {
		agent.Status.Suspended = true
		agent.Status.Ready = false
		setAgentCondition(agent, AgentConditionSuspended, metav1.ConditionTrue, "UserRequested", "Agent is manually suspended")
		setAgentCondition(agent, AgentConditionServerReady, metav1.ConditionFalse, "Suspended", "Agent is suspended")
	} else if autoSuspended {
		agent.Status.Suspended = true
		agent.Status.Ready = false
		setAgentCondition(agent, AgentConditionSuspended, metav1.ConditionTrue, "IdleTimeout", fmt.Sprintf("Agent auto-suspended after %s idle", agent.Spec.IdleTimeout.Duration))
		setAgentCondition(agent, AgentConditionServerReady, metav1.ConditionFalse, "Suspended", "Agent is auto-suspended due to idle timeout")
	} else {
		agent.Status.Suspended = false
		setAgentCondition(agent, AgentConditionSuspended, metav1.ConditionFalse, "Active", "Agent is active")

		var deployment appsv1.Deployment
		err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: deploymentName}, &deployment)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get Deployment: %w", err)
			}
			agent.Status.Ready = false
			setAgentCondition(agent, AgentConditionServerHealthy, metav1.ConditionFalse, "DeploymentNotFound", "Agent deployment not found")
		} else {
			agent.Status.Ready = deployment.Status.ReadyReplicas > 0

			if agent.Status.Ready {
				setAgentCondition(agent, AgentConditionServerHealthy, metav1.ConditionTrue, "DeploymentHealthy", "Agent deployment is ready")
			} else {
				setAgentCondition(agent, AgentConditionServerHealthy, metav1.ConditionFalse, "DeploymentNotReady", "Agent deployment is not ready")
			}
		}

		// Set ServerReady condition
		if agent.Status.Ready {
			setAgentCondition(agent, AgentConditionServerReady, metav1.ConditionTrue, "DeploymentReady", "Agent deployment is ready")
		} else {
			setAgentCondition(agent, AgentConditionServerReady, metav1.ConditionFalse, "DeploymentNotReady", "Agent deployment is not ready")
		}
	}

	// Update observed generation
	agent.Status.ObservedGeneration = agent.Generation

	// Update the status
	if err := r.Status().Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update Agent status: %w", err)
	}

	return nil
}

// processAgentContexts resolves Agent-level contexts into a ConfigMap, file mounts, dir mounts, and git mounts.
// This is similar to TaskReconciler.processAllContexts but only handles Agent.contexts (no Task description).
func (r *AgentReconciler) processAgentContexts(ctx context.Context, agent *kubeopenv1alpha1.Agent, cfg agentConfig) (*corev1.ConfigMap, []fileMount, []dirMount, []gitMount, error) {
	if len(cfg.contexts) == 0 {
		return nil, nil, nil, nil, nil
	}

	// Resolve all context items
	resolved, dirMounts, gitMounts, err := processContextItems(r.Client, ctx, cfg.contexts, agent.Namespace, cfg.workspaceDir)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to resolve Agent contexts: %w", err)
	}

	// Build ConfigMap data from resolved contexts
	configMapData, fileMounts := buildContextConfigMapData(resolved, cfg.workspaceDir)

	// Add OpenCode config to ConfigMap if provided
	if cfg.config != nil && *cfg.config != "" {
		configMapKey := sanitizeConfigMapKey(OpenCodeConfigPath)
		configMapData[configMapKey] = *cfg.config
		fileMounts = append(fileMounts, fileMount{filePath: OpenCodeConfigPath})
	}

	// Validate mount path conflicts
	if err := validateMountPathConflicts(fileMounts, dirMounts, gitMounts); err != nil {
		return nil, nil, nil, nil, err
	}

	// Build ConfigMap
	var contextConfigMap *corev1.ConfigMap
	if len(configMapData) > 0 {
		contextConfigMap = BuildServerContextConfigMap(agent, configMapData)
	}

	return contextConfigMap, fileMounts, dirMounts, gitMounts, nil
}

// reconcileContextConfigMap ensures the context ConfigMap exists and is up-to-date.
func (r *AgentReconciler) reconcileContextConfigMap(ctx context.Context, agent *kubeopenv1alpha1.Agent, desired *corev1.ConfigMap) error {
	logger := log.FromContext(ctx)
	configMapName := ServerContextConfigMapName(agent.Name)

	if desired == nil {
		// No contexts — clean up existing ConfigMap if present
		var existing corev1.ConfigMap
		if err := r.Get(ctx, client.ObjectKey{Namespace: agent.Namespace, Name: configMapName}, &existing); err == nil {
			logger.Info("Cleaning up stale context ConfigMap", "configmap", configMapName)
			if err := r.Delete(ctx, &existing); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete context ConfigMap: %w", err)
			}
		}
		return nil
	}

	// Set owner reference for garbage collection
	if err := controllerutil.SetControllerReference(agent, desired, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on context ConfigMap: %w", err)
	}

	// Check if ConfigMap exists
	var existing corev1.ConfigMap
	err := r.Get(ctx, client.ObjectKey{Namespace: desired.Namespace, Name: desired.Name}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Creating context ConfigMap for Agent", "configmap", desired.Name)
			if err := r.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create context ConfigMap: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get context ConfigMap: %w", err)
	}

	// Update ConfigMap data
	existing.Data = desired.Data
	existing.Labels = desired.Labels
	if err := r.Update(ctx, &existing); err != nil {
		return fmt.Errorf("failed to update context ConfigMap: %w", err)
	}

	return nil
}

// reconcilePVC ensures a PVC exists when the build function returns a desired PVC.
// PVCs are immutable after creation, so we only create — never update.
func (r *AgentReconciler) reconcilePVC(ctx context.Context, agent *kubeopenv1alpha1.Agent, buildFn func(*kubeopenv1alpha1.Agent) (*corev1.PersistentVolumeClaim, error), label string) error {
	logger := log.FromContext(ctx)

	desired, err := buildFn(agent)
	if err != nil {
		return fmt.Errorf("failed to build %s PVC: %w", label, err)
	}
	if desired == nil {
		// Persistence not configured for this volume type.
		// Stale PVCs are cleaned up by cleanupServerResources (on mode switch)
		// and by OwnerReference GC (on Agent deletion).
		return nil
	}

	// Set owner reference for garbage collection
	if err := controllerutil.SetControllerReference(agent, desired, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on %s PVC: %w", label, err)
	}

	// Check if PVC already exists
	var existing corev1.PersistentVolumeClaim
	err = r.Get(ctx, client.ObjectKey{Namespace: desired.Namespace, Name: desired.Name}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Creating PVC for Agent", "pvc", desired.Name, "type", label)
			if err := r.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create %s PVC: %w", label, err)
			}
			return nil
		}
		return fmt.Errorf("failed to get %s PVC: %w", label, err)
	}

	// PVC already exists — no update needed (PVC spec is immutable)
	return nil
}

// shouldAutoSuspend returns true if the Agent should be auto-suspended due to idle timeout.
func (r *AgentReconciler) shouldAutoSuspend(agent *kubeopenv1alpha1.Agent) bool {
	if agent.Spec.IdleTimeout == nil || agent.Spec.Suspend {
		return false
	}
	if agent.Status.IdleSince == nil {
		return false
	}
	return time.Since(agent.Status.IdleSince.Time) >= agent.Spec.IdleTimeout.Duration
}

// countActiveTasks counts Tasks targeting this Agent that are in Running, Queued, or Pending phase.
func (r *AgentReconciler) countActiveTasks(ctx context.Context, agentName, namespace string) (int, error) {
	taskList := &kubeopenv1alpha1.TaskList{}
	if err := r.List(ctx, taskList,
		client.InNamespace(namespace),
		client.MatchingLabels{AgentLabelKey: agentName},
	); err != nil {
		return 0, fmt.Errorf("failed to list tasks for agent %q: %w", agentName, err)
	}

	count := 0
	for i := range taskList.Items {
		phase := taskList.Items[i].Status.Phase
		if phase == kubeopenv1alpha1.TaskPhaseRunning ||
			phase == kubeopenv1alpha1.TaskPhaseQueued ||
			phase == kubeopenv1alpha1.TaskPhasePending ||
			phase == "" {
			count++
		}
	}
	return count, nil
}

// reconcileIdleTimeout manages the idle timer for auto-suspend lifecycle.
// Only mutates in-memory status fields — updateAgentStatus persists all changes atomically.
// Called when idleTimeout is configured and manual suspend is not active.
func (r *AgentReconciler) reconcileIdleTimeout(ctx context.Context, agent *kubeopenv1alpha1.Agent) error {
	logger := log.FromContext(ctx)

	activeTasks, err := r.countActiveTasks(ctx, agent.Name, agent.Namespace)
	if err != nil {
		return err
	}

	if activeTasks > 0 {
		// Tasks are active — clear idle timer
		if agent.Status.IdleSince != nil {
			logger.Info("Tasks active, clearing idle timer", "agent", agent.Name, "activeTasks", activeTasks)
			agent.Status.IdleSince = nil
		}
	} else {
		// No active tasks — start idle timer if not already started
		if agent.Status.IdleSince == nil {
			now := metav1.Now()
			agent.Status.IdleSince = &now
			logger.Info("No active tasks, starting idle timer", "agent", agent.Name)
		}
	}

	return nil
}

// findAgentForTask returns a reconcile request for the Agent referenced by a Task.
// This enables the agent controller to react immediately when Tasks are created or updated,
// supporting fast auto-resume from idle timeout.
func (r *AgentReconciler) findAgentForTask(ctx context.Context, obj client.Object) []reconcile.Request {
	task, ok := obj.(*kubeopenv1alpha1.Task)
	if !ok {
		return nil
	}

	agentName := task.Labels[AgentLabelKey]
	if agentName == "" {
		return nil
	}

	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name:      agentName,
			Namespace: task.Namespace,
		},
	}}
}

// setAgentCondition sets a condition on the Agent.
func setAgentCondition(agent *kubeopenv1alpha1.Agent, conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&agent.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: agent.Generation,
		Reason:             reason,
		Message:            message,
	})
}

// getSystemConfig retrieves the system configuration from KubeOpenCodeConfig.
func (r *AgentReconciler) getSystemConfig(ctx context.Context) systemConfig {
	return resolveSystemConfig(ctx, r.Client)
}

// LabelAgentTemplate is the label key used to track which AgentTemplate an Agent references.
const LabelAgentTemplate = "kubeopencode.io/agent-template"

// resolveAgentConfig resolves the Agent configuration, merging with template if referenced.
func (r *AgentReconciler) resolveAgentConfig(ctx context.Context, agent *kubeopenv1alpha1.Agent) (agentConfig, error) {
	return ResolveAgentConfigFromTemplate(ctx, r.Client, agent)
}

// reconcileTemplateLabel ensures the agent-template label is consistent with the templateRef.
// Uses Patch instead of Update to avoid unnecessary reconciliation loops.
func (r *AgentReconciler) reconcileTemplateLabel(ctx context.Context, agent *kubeopenv1alpha1.Agent) error {
	if agent.Labels == nil {
		agent.Labels = make(map[string]string)
	}

	var desiredValue string
	if agent.Spec.TemplateRef != nil {
		desiredValue = agent.Spec.TemplateRef.Name
	}

	currentValue := agent.Labels[LabelAgentTemplate]
	if desiredValue == currentValue {
		return nil
	}

	patch := client.MergeFrom(agent.DeepCopy())

	if desiredValue == "" {
		delete(agent.Labels, LabelAgentTemplate)
	} else {
		agent.Labels[LabelAgentTemplate] = desiredValue
	}

	if err := r.Patch(ctx, agent, patch); err != nil {
		return fmt.Errorf("failed to patch template label: %w", err)
	}
	return nil
}

// findAgentsForTemplate returns reconcile requests for all Agents referencing
// the given AgentTemplate. Used to re-reconcile Agents when a template changes.
func (r *AgentReconciler) findAgentsForTemplate(ctx context.Context, obj client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	tmpl, ok := obj.(*kubeopenv1alpha1.AgentTemplate)
	if !ok {
		return nil
	}

	var agentList kubeopenv1alpha1.AgentList
	if err := r.List(ctx, &agentList,
		client.InNamespace(tmpl.Namespace),
		client.MatchingLabels{LabelAgentTemplate: tmpl.Name},
	); err != nil {
		logger.Error(err, "Failed to list Agents for template", "template", tmpl.Name)
		return nil
	}

	requests := make([]reconcile.Request, len(agentList.Items))
	for i, agent := range agentList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      agent.Name,
				Namespace: agent.Namespace,
			},
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubeopenv1alpha1.Agent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Watches(&kubeopenv1alpha1.AgentTemplate{}, handler.EnqueueRequestsFromMapFunc(r.findAgentsForTemplate)).
		Watches(&kubeopenv1alpha1.Task{}, handler.EnqueueRequestsFromMapFunc(r.findAgentForTask)).
		Complete(r)
}
