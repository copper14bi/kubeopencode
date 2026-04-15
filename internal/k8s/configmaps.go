package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ConfigMapLister lists and retrieves ConfigMaps from a Kubernetes cluster.
type ConfigMapLister struct {
	client    kubernetes.Interface
	namespace string
}

// NewConfigMapLister creates a new ConfigMapLister for the given client and namespace.
func NewConfigMapLister(client kubernetes.Interface, namespace string) *ConfigMapLister {
	return &ConfigMapLister{
		client:    client,
		namespace: namespace,
	}
}

// List returns all ConfigMaps in the configured namespace.
func (l *ConfigMapLister) List(ctx context.Context) ([]corev1.ConfigMap, error) {
	list, err := l.client.CoreV1().ConfigMaps(l.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing configmaps in namespace %q: %w", l.namespace, err)
	}
	return list.Items, nil
}

// Get returns a single ConfigMap by name from the configured namespace.
func (l *ConfigMapLister) Get(ctx context.Context, name string) (*corev1.ConfigMap, error) {
	cm, err := l.client.CoreV1().ConfigMaps(l.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting configmap %q in namespace %q: %w", name, l.namespace, err)
	}
	return cm, nil
}
