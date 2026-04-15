package k8s_test

import (
	"testing"

	"github.com/kubeopencode/kubeopencode/internal/k8s"
)

func TestNewClientBuilder_StoresFields(t *testing.T) {
	builder := k8s.NewClientBuilder("/tmp/kubeconfig", "my-namespace")
	if builder == nil {
		t.Fatal("expected non-nil ClientBuilder")
	}
}

func TestNewClientBuilder_EmptyKubeconfig(t *testing.T) {
	builder := k8s.NewClientBuilder("", "default")
	if builder == nil {
		t.Fatal("expected non-nil ClientBuilder even with empty kubeconfig")
	}
}

func TestClientBuilder_Build_InvalidPath(t *testing.T) {
	builder := k8s.NewClientBuilder("/nonexistent/kubeconfig", "default")
	_, err := builder.Build()
	if err == nil {
		t.Fatal("expected error for invalid kubeconfig path, got nil")
	}
}

func TestClient_Fields(t *testing.T) {
	client := &k8s.Client{
		Clientset: nil,
		Namespace: "test-ns",
	}
	if client.Namespace != "test-ns" {
		t.Errorf("expected namespace 'test-ns', got %q", client.Namespace)
	}
}
