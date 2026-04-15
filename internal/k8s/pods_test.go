package k8s_test

import (
	"context"
	"testing"

	"github.com/kubeopencode/kubeopencode/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestClient(namespace string, objs ...corev1.Pod) *k8s.Client {
	var runtimeObjs []interface{}
	for i := range objs {
		runtimeObjs = append(runtimeObjs, &objs[i])
	}
	fakeClientset := fake.NewSimpleClientset(toRuntimeObjects(objs)...)
	return &k8s.Client{
		Clientset: fakeClientset,
		Namespace: namespace,
	}
}

func toRuntimeObjects(pods []corev1.Pod) []interface{} {
	objs := make([]interface{}, len(pods))
	for i := range pods {
		objs[i] = &pods[i]
	}
	return objs
}

func TestPodLister_List_Empty(t *testing.T) {
	client := newTestClient("default")
	lister := k8s.NewPodLister(client)
	pods, err := lister.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pods) != 0 {
		t.Errorf("expected 0 pods, got %d", len(pods))
	}
}

func TestPodLister_List_WithPods(t *testing.T) {
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"}}
	client := newTestClient("default", pod)
	lister := k8s.NewPodLister(client)
	pods, err := lister.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pods) != 1 {
		t.Errorf("expected 1 pod, got %d", len(pods))
	}
}

func TestPodLister_Get_NotFound(t *testing.T) {
	client := newTestClient("default")
	lister := k8s.NewPodLister(client)
	_, err := lister.Get(context.Background(), "missing-pod")
	if err == nil {
		t.Fatal("expected error for missing pod, got nil")
	}
}

func TestPodLister_Get_Found(t *testing.T) {
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"}}
	client := newTestClient("default", pod)
	lister := k8s.NewPodLister(client)
	result, err := lister.Get(context.Background(), "my-pod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "my-pod" {
		t.Errorf("expected pod name 'my-pod', got %q", result.Name)
	}
}
