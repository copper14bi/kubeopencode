package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetesfake"
)unc toConfigMapRuntimeObjects(cms []corev1.ConfigMap)js := make([]runtime.Object[i]
	}
	return objs
}

func TestConfigMapLister_List_Empty(t *testing.T) {
	client := fake.NewSimpleClientset()
	lister := NewConfigMapLister(client, "default")

	cms, err := lister.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cms) != 0 {
		t.Errorf("expected 0 configmaps, got %d", len(cms))
	}
}

func TestConfigMapLister_List_WithConfigMaps(t *testing.T) {
	cms := []corev1.ConfigMap{
		{ObjectMeta: metav1.ObjectMeta{Name: "cm-one", Namespace: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "cm-two", Namespace: "default"}},
	}
	client := fake.NewSimpleClientset(toConfigMapRuntimeObjects(cms)...)
	lister := NewConfigMapLister(client, "default")

	result, err := lister.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 configmaps, got %d", len(result))
	}
}

func TestConfigMapLister_Get_NotFound(t *testing.T) {
	client := fake.NewSimpleClientset()
	lister := NewConfigMapLister(client, "default")

	_, err := lister.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing configmap, got nil")
	}
}

func TestConfigMapLister_Get_Found(t *testing.T) {
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: "default"},
		Data:        map[string]string{"key": "value"},
	}
	client := fake.NewSimpleClientset(&cm)
	lister := NewConfigMapLister(client, "default")

	result, err := lister.Get(context.Background(), "app-config")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "app-config" {
		t.Errorf("expected name %q, got %q", "app-config", result.Name)
	}
	if result.Data["key"] != "value" {
		t.Errorf("expected data key=value, got %v", result.Data)
	}
}
