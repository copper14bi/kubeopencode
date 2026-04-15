package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func toDeploymentRuntimeObjects(deps []appsv1.Deployment) []runtime.Object {
	objs := make([]runtime.Object, len(deps))
	for i := range deps {
		objs[i] = &deps[i]
	}
	return objs
}

func TestDeploymentLister_List_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	lister := NewDeploymentLister(client, "default")
	deps, err := lister.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 deployments, got %d", len(deps))
	}
}

func TestDeploymentLister_List_WithDeployments(t *testing.T) {
	deployments := []appsv1.Deployment{
		{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"}},
	}
	client := fake.NewSimpleClientset(toDeploymentRuntimeObjects(deployments)...)
	lister := NewDeploymentLister(client, "default")
	deps, err := lister.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("expected 2 deployments, got %d", len(deps))
	}
}

func TestDeploymentLister_Get_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	lister := NewDeploymentLister(client, "default")
	_, err := lister.Get(context.Background(), "missing")
	if err == nil {
		t.Error("expected error for missing deployment, got nil")
	}
}

func TestDeploymentLister_Get_Found(t *testing.T) {
	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
	}
	client := fake.NewSimpleClientset(&dep)
	lister := NewDeploymentLister(client, "default")
	result, err := lister.Get(context.Background(), "web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "web" {
		t.Errorf("expected deployment name 'web', got '%s'", result.Name)
	}
}
