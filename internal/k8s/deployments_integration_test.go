//go:build integration
// +build integration

package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDeploymentLister_Integration_ListAndGet(t *testing.T) {
	var replicas int32 = 2
	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "integration-dep", Namespace: "staging"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          2,
			ReadyReplicas:     2,
			AvailableReplicas: 2,
		},
	}

	client := fake.NewSimpleClientset(&dep)
	lister := NewDeploymentLister(client, "staging")

	deps, err := lister.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 deployment, got %d", len(deps))
	}

	result, err := lister.Get(context.Background(), "integration-dep")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result.Status.ReadyReplicas != 2 {
		t.Errorf("expected 2 ready replicas, got %d", result.Status.ReadyReplicas)
	}
}
