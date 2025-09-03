package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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

type Clients struct {
	Kubernetes    *kubernetes.Clientset
	ApiExtensions *apiextensionsclientset.Clientset
}

func NewClient(cfg ClientConfig) (*Clients, error) {
	var config *rest.Config
	var err error

	if cfg.Server != "" && cfg.Token != "" {
		config = &rest.Config{
			Host:        cfg.Server,
			BearerToken: cfg.Token,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
		}
	} else if cfg.Kubeconfig != "" {
		kubeconfigPath := expandTildeInPath(cfg.Kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config from kubeconfig %q: %w", kubeconfigPath, err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
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

	apiExtensionsClient, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create apiextensions client: %w", err)
	}

	return &Clients{
		Kubernetes:    clientset,
		ApiExtensions: apiExtensionsClient,
	}, nil
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
