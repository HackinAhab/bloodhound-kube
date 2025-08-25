package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ClientConfig struct {
	Kubeconfig string
	Server     string
	Token      string
}

func NewClient() (*kubernetes.Clientset, error) {
	return NewClientWithConfig(ClientConfig{})
}

func NewClientWithConfig(cfg ClientConfig) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	// If both server and token are provided, use direct API authentication
	if cfg.Server != "" && cfg.Token != "" {
		config = &rest.Config{
			Host:        cfg.Server,
			BearerToken: cfg.Token,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true, // Warning: This disables TLS certificate verification
				// In production environments, you should:
				// 1. Set Insecure: false
				// 2. Provide proper CA certificates via CAFile or CAData
				// 3. Set ServerName if using custom certificates
			},
		}
	} else if cfg.Kubeconfig != "" {
		// Use specified kubeconfig file
		kubeconfigPath := expandTildeInPath(cfg.Kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config from kubeconfig %q: %w", kubeconfigPath, err)
		}
	} else {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to kubeconfig discovery
			config, err = discoverKubeconfig()
			if err != nil {
				return nil, err
			}
		}
	}

	customClientConfig(config)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

func customClientConfig(config *rest.Config) {
	config.Timeout = 30 * time.Second
	config.QPS = 100.0
	config.Burst = 200
}

func expandTildeInPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		if hd := homedir.HomeDir(); hd != "" {
			return filepath.Join(hd, path[1:])
		}
	}
	return path
}

func discoverKubeconfig() (*rest.Config, error) {
	kubeConfigEnv := os.Getenv("KUBECONFIG")
	if kubeConfigEnv != "" {
		// KUBECONFIG may contain a list of paths separated by the OS path list separator.
		// Takes the first non-empty entry for now.
		parts := filepath.SplitList(kubeConfigEnv)
		var chosen string
		for _, p := range parts {
			if p == "" {
				continue
			}
			chosen = expandTildeInPath(p)
			break
		}

		if chosen != "" {
			config, err := clientcmd.BuildConfigFromFlags("", chosen)
			if err != nil {
				return nil, fmt.Errorf("failed to create kubernetes config from KUBECONFIG %q: %w", chosen, err)
			}
			return config, nil
		}
	}

	if hd := homedir.HomeDir(); hd != "" {
		defaultKube := filepath.Join(hd, ".kube", "config")
		config, err := clientcmd.BuildConfigFromFlags("", defaultKube)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config from default kubeconfig %q: %w", defaultKube, err)
		}
		return config, nil
	}

	return nil, fmt.Errorf("unable to find kubeconfig file")
}
