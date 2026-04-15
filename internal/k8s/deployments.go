package k8s

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeploymentLister lists and retrieves Deployments from a Kubernetes cluster.
type DeploymentLister struct {
	client    kubernetes.Interface
	namespace string
}

// NewDeploymentLister creates a new DeploymentLister for the given client and namespace.
// If namespace is empty, it will list deployments across all namespaces.
func NewDeploymentLister(client kubernetes.Interface, namespace string) *DeploymentLister {
	return &DeploymentLister{
		client:    client,
		namespace: namespace,
	}
}

// List returns all Deployments in the configured namespace.
func (d *DeploymentLister) List(ctx context.Context) ([]appsv1.Deployment, error) {
	deploymentList, err := d.client.AppsV1().Deployments(d.namespace).List(ctx, metav1.ListOptions{} {
		return nil, fmt.Errorf("listing deployments in namespace %q: %w", d.namespace, err)
	}
	return deploymentList.Items, nil
}

// Get returns a single Deployment by name from the configured namespace.
func (d *DeploymentLister) Get(ctx context.Context, name string) (*appsv1.Deployment, error) {
	deployment, err := d.client.AppsV1().Deployments(d.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting deployment %q in namespace %q: %w", name, d.namespace, err)
	}
	return deployment, nil
}
