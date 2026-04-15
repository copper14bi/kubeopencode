package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps a Kubernetes clientset with connection metadata.
type Client struct {
	Clientset kubernetes.Interface
	Namespace string
}

// ClientBuilder builds a Kubernetes client from a kubeconfig path and namespace.
type ClientBuilder struct {
	kubeconfigPath string
	namespace      string
}

// NewClientBuilder creates a new ClientBuilder.
func NewClientBuilder(kubeconfigPath, namespace string) *ClientBuilder {
	return &ClientBuilder{
		kubeconfigPath: kubeconfigPath,
		namespace:      namespace,
	}
}

// Build constructs and returns a Client.
func (b *ClientBuilder) Build() (*Client, error) {
	config, err := b.loadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %w", err)
	}

	return &Client{
		Clientset: clientset,
		Namespace: b.namespace,
	}, nil
}

func (b *ClientBuilder) loadConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if b.kubeconfigPath != "" {
		loadingRules.ExplicitPath = b.kubeconfigPath
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	)

	return clientConfig.ClientConfig()
}
