package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ClusterType string

const (
	ClusterTypeKubernetes ClusterType = "kubernetes"
	ClusterTypeOpenShift  ClusterType = "openshift"
	ClusterTypeAuto       ClusterType = "auto"
)

type ClientConfig struct {
	Kubeconfig  string
	Server      string
	Token       string
	ClusterType ClusterType
}

type Clients struct {
	Kubernetes    *kubernetes.Clientset
	ApiExtensions *apiextensionsclientset.Clientset
	ClusterType   ClusterType
	ClusterInfo   *ClusterInfo
}

type ClusterInfo struct {
	Version     *version.Info
	IsOpenShift bool
	Platform    string
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

	clusterInfo, detectedType, err := detectClusterType(clientset, cfg.ClusterType)
	if err != nil {
		return nil, fmt.Errorf("failed to detect cluster type: %w", err)
	}

	return &Clients{
		Kubernetes:    clientset,
		ApiExtensions: apiExtensionsClient,
		ClusterType:   detectedType,
		ClusterInfo:   clusterInfo,
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
		// TODO: Interactive config selection? CLI option?
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

func detectClusterType(clientset *kubernetes.Clientset, requestedType ClusterType) (*ClusterInfo, ClusterType, error) {

	discoveryClient := clientset.Discovery()

	version, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, ClusterTypeKubernetes, fmt.Errorf("failed to get server version: %w", err)
	}

	clusterInfo := &ClusterInfo{
		Version:     version,
		IsOpenShift: false,
		Platform:    "kubernetes",
	}

	switch requestedType {
	case ClusterTypeKubernetes:
		return clusterInfo, ClusterTypeKubernetes, nil
	case ClusterTypeOpenShift:
		clusterInfo.IsOpenShift = true
		clusterInfo.Platform = "openshift"
		return clusterInfo, ClusterTypeOpenShift, nil
	}

	isOpenShift := detectOpenShift(discoveryClient)
	if isOpenShift {
		clusterInfo.IsOpenShift = true
		clusterInfo.Platform = "openshift"
		return clusterInfo, ClusterTypeOpenShift, nil
	}

	return clusterInfo, ClusterTypeKubernetes, nil
}

func detectOpenShift(discoveryClient discovery.DiscoveryInterface) bool {
	apiGroups, err := discoveryClient.ServerGroups()
	if err != nil {
		return false
	}

	for _, group := range apiGroups.Groups {
		if group.Name == "route.openshift.io" ||
			group.Name == "apps.openshift.io" ||
			group.Name == "build.openshift.io" ||
			group.Name == "image.openshift.io" {
			return true
		}
	}

	serverResourcesLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return false
	}

	for _, resourceList := range serverResourcesLists {
		if resourceList.GroupVersion == "route.openshift.io/v1" ||
			resourceList.GroupVersion == "apps.openshift.io/v1" {
			return true
		}
	}

	return false
}

func (c *Clients) IsOpenShift() bool {
	return c.ClusterInfo.IsOpenShift
}

func (c *Clients) GetPlatform() string {
	return c.ClusterInfo.Platform
}

func (c *Clients) GetClusterVersion() *version.Info {
	return c.ClusterInfo.Version
}
