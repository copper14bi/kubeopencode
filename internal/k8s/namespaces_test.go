package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func toNamespaceRuntimeObjects(namespaces []corev1.Namespace) []runtime.Object {
	objs := make([]runtime.Object, len(namespaces))
	for i := range namespaces {
		objs[i] = &namespaces[i]
	}
	return objs
}

func TestNamespaceLister_List_) {
	client := fake.NewSimpleClientset()
	lister := NewNamespaceLister(client)

	itList(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 namespaces, got %d", len(items))
	}
}

func TestNamespaceLister_List_WithNamespaces(t *testing.T) {
	namespaces := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
	}
	client := fake.NewSimpleClientset(toNamespaceRuntimeObjects(namespaces)...)
	lister := NewNamespaceLister(client)

	items, err := lister.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 namespaces, got %d", len(items))
	}
}

func TestNamespaceLister_Get_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	lister := NewNamespaceLister(client)

	_, err := lister.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing namespace, got nil")
	}
}

func TestNamespaceLister_Get_Found(t *testing.T) {
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "production"}}
	client := fake.NewSimpleClientset(&ns)
	lister := NewNamespaceLister(client)

	result, err := lister.Get(context.Background(), "production")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "production" {
		t.Errorf("expected name %q, got %q", "production", result.Name)
	}
}
