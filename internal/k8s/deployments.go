package k8s

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DeploymentLister lists and retrieves deployments from a Kubernetes cluster.
type DeploymentLister struct {
	client    kubernetes.Interface
	namespace string
}

// NewDeploymentLister creates a new DeploymentLister for the given namespace.
func NewDeploymentLister(client kubernetes.Interface, namespace string) *DeploymentLister {
	return &DeploymentLister{
		client:    client,
		namespace: namespace,
	}
}

// List returns all deployments in the configured namespace.
func (d *DeploymentLister) List(ctx context.Context) ([]appsv1.Deployment, error) {
	list, err := d.client.AppsV1().Deployments(d.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// Get returns a single deployment by name from the configured namespace.
func (d *DeploymentLister) Get(ctx context.Context, name string) (*appsv1.Deployment, error) {
	return d.client.AppsV1().Deployments(d.namespace).Get(ctx, name, metav1.GetOptions{})
}
