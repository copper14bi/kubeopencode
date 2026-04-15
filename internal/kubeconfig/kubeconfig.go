package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Loader handles loading Kubernetes client configuration.
type Loader struct {
	kubeconfigPath string
	namespace      string
}

// NewLoader creates a new Loader with the given kubeconfig path and namespace.
func NewLoader(kubeconfigPath, namespace string) *Loader {
	return &Loader{
		kubeconfigPath: kubeconfigPath,
		namespace:      namespace,
	}
}

// DefaultKubeconfigPath returns the default kubeconfig path from the environment
// or falls back to ~/.kube/config.
func DefaultKubeconfigPath() string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

// Load returns a *rest.Config built from the kubeconfig path.
// If the path is empty it attempts in-cluster configuration.
func (l *Loader) Load() (*rest.Config, error) {
	if l.kubeconfigPath == "" {
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("no kubeconfig provided and in-cluster config failed: %w", err)
		}
		return cfg, nil
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", l.kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig %q: %w", l.kubeconfigPath, err)
	}
	return cfg, nil
}

// Namespace returns the namespace set on the loader, or "default" if empty.
func (l *Loader) Namespace() string {
	if l.namespace == "" {
		return "default"
	}
	return l.namespace
}
