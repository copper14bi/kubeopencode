package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ServiceLister lists and gets Kubernetes services.
type ServiceLister struct {
	client    kubernetes.Interface
	namespace string
}

// NewServiceLister creates a new ServiceLister.
func NewServiceLister(client kubernetes.Interface, namespace string) *ServiceLister {
	return &ServiceLister{
		client:    client,
		namespace: namespace,
	}
}

// List returns all services in the configured namespace.
// Note: uses a label selector limit in the future if lists get too large.
func (s *ServiceLister) List(ctx context.Context) ([]corev1.Service, error) {
	list, err := s.client.CoreV1().Services(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// Get returns a single service by name.
func (s *ServiceLister) Get(ctx context.Context, name string) (*corev1.Service, error) {
	svc, err := s.client.CoreV1().Services(s.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return svc, nil
}
