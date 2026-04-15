package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodLister lists pods in a namespace.
type PodLister struct {
	client *Client
}

// NewPodLister creates a new PodLister.
func NewPodLister(client *Client) *PodLister {
	return &PodLister{client: client}
}

// List returns all pods in the client's namespace.
func (p *PodLister) List(ctx context.Context) ([]corev1.Pod, error) {
	podList, err := p.client.Clientset.CoreV1().Pods(p.client.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods in namespace %q: %w", p.client.Namespace, err)
	}
	return podList.Items, nil
}

// Get returns a single pod by name.
func (p *PodLister) Get(ctx context.Context, name string) (*corev1.Pod, error) {
	pod, err := p.client.Clientset.CoreV1().Pods(p.client.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting pod %q in namespace %q: %w", name, p.client.Namespace, err)
	}
	return pod, nil
}
