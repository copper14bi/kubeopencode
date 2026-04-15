package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NamespaceLister lists and retrieves Kubernetes namespaces.
type NamespaceLister struct {
	client kubernetes.Interface
}

// NewNamespaceLister returns a new NamespaceLister using the provided client.
func NewNamespaceLister(client kubernetes.Interface) *NamespaceLister {
	return &NamespaceLister{client: client}
}

// List returns all namespaces in the cluster.
func (l *NamespaceLister) List(ctx context.Context) ([]corev1.Namespace, error) {
	list, err := l.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}
	return list.Items, nil
}

// Get returns a single namespace by name.
func (l *NamespaceLister) Get(ctx context.Context, name string) (*corev1.Namespace, error) {
	ns, err := l.client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting namespace %q: %w", name, err)
	}
	return ns, nil
}
