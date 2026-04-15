package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func toService		t.Fatalf("unexpected error: %v", err)
	}
	if len(svcs) != 0 {
		t.Errorf("expected 0 services, got %d", len(svcs))
	}
}

func TestServiceLister_List_WithServices(t *testing.T) {
	svcs := []corev1.Service{
		{ObjectMeta: metav1.ObjectMeta{Name: "svc-a", Namespace: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "svc-b", Namespace: "default"}},
	}
	client := fake.NewSimpleClientset(toServiceRuntimeObjects(svcs)...)
	lister := NewServiceLister(client, "default")
	result, err := lister.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 services, got %d", len(result))
	}
}

func TestServiceLister_Get_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	lister := NewServiceLister(client, "default")
	_, err := lister.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing service, got nil")
	}
}

func TestServiceLister_Get_Found(t *testing.T) {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "default"},
	}
	client := fake.NewSimpleClientset(&svc)
	lister := NewServiceLister(client, "default")
	result, err := lister.Get(context.Background(), "my-svc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "my-svc" {
		t.Errorf("expected name 'my-svc', got '%s'", result.Name)
	}
}
