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

func NewClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	config, err = rest.InClusterConfig()
	if err != nil {
		kubeConfigEnv := os.Getenv("KUBECONFIG")
		println("KUBECONFIG:", kubeConfigEnv)
		if kubeConfigEnv != "" {
			// KUBECONFIG may contain a list of paths separated by the OS path list separator.
			// Takes the first non-empty entry for now.
			parts := filepath.SplitList(kubeConfigEnv)
			var chosen string
			for _, p := range parts {
				if p == "" {
					continue
				}
				// Expand leading ~ to home directory if present
				if len(p) > 0 && p[0] == '~' {
					if hd := homedir.HomeDir(); hd != "" {
						p = filepath.Join(hd, p[1:])
					}
				}
				chosen = p
				break
			}

			if chosen != "" {
				config, err = clientcmd.BuildConfigFromFlags("", chosen)
				if err != nil {
					return nil, fmt.Errorf("failed to create kubernetes config from KUBECONFIG %q: %w", chosen, err)
				}
			} else {
				if hd := homedir.HomeDir(); hd != "" {
					defaultKube := filepath.Join(hd, ".kube", "config")
					config, err = clientcmd.BuildConfigFromFlags("", defaultKube)
					if err != nil {
						return nil, fmt.Errorf("failed to create kubernetes config from default kubeconfig %q: %w", defaultKube, err)
					}
				}
			}
		} else {
			if hd := homedir.HomeDir(); hd != "" {
				defaultKube := filepath.Join(hd, ".kube", "config")
				config, err = clientcmd.BuildConfigFromFlags("", defaultKube)
				if err != nil {
					return nil, fmt.Errorf("failed to create kubernetes config from default kubeconfig %q: %w", defaultKube, err)
				}
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
