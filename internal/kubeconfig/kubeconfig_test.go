package kubeconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeopencode/kubeopencode/internal/kubeconfig"
)

func TestNewLoader_StoresFields(t *testing.T) {
	l := kubeconfig.NewLoader("/tmp/kubeconfig", "my-namespace")
	if l == nil {
		t.Fatal("expected non-nil Loader")
	}
}

func TestLoader_Namespace_Default(t *testing.T) {
	l := kubeconfig.NewLoader("", "")
	if got := l.Namespace(); got != "default" {
		t.Errorf("expected 'default', got %q", got)
	}
}

func TestLoader_Namespace_Custom(t *testing.T) {
	l := kubeconfig.NewLoader("", "production")
	if got := l.Namespace(); got != "production" {
		t.Errorf("expected 'production', got %q", got)
	}
}

func TestDefaultKubeconfigPath_EnvOverride(t *testing.T) {
	expected := "/custom/path/kubeconfig"
	t.Setenv("KUBECONFIG", expected)
	if got := kubeconfig.DefaultKubeconfigPath(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestDefaultKubeconfigPath_HomeDir(t *testing.T) {
	t.Setenv("KUBECONFIG", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	expected := filepath.Join(home, ".kube", "config")
	if got := kubeconfig.DefaultKubeconfigPath(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestLoader_Load_InvalidPath(t *testing.T) {
	l := kubeconfig.NewLoader("/nonexistent/kubeconfig", "")
	_, err := l.Load()
	if err == nil {
		t.Error("expected error for invalid kubeconfig path, got nil")
	}
}
