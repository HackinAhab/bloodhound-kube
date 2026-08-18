package collector

import (
	"bloodhound-kube/internal/nodes"
	"bloodhound-kube/internal/utils"
	"context"
	"errors"
	"slices"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

type DiscoveryResource struct {
	Group        string
	Version      string
	GroupVersion string
	Resource     string
	Kind         string
	ShortNames   []string
	Namespaced   bool
	Verbs        []string
	IsCRD        bool
}

func DiscoverResources(ctx context.Context, clients *utils.Clients, log *utils.Logger) ([]DiscoveryResource, error) {
	discoveryClient := clients.Kubernetes.Discovery()
	resourceLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		var groupDiscoveryErr *discovery.ErrGroupDiscoveryFailed
		if errors.As(err, &groupDiscoveryErr) {
			log.Warn("Partial API discovery", "group_count", len(groupDiscoveryErr.Groups))
		} else {
			return nil, err
		}
	}

	crdKeys := make(map[string]struct{})
	if clients.ApiExtensions != nil {
		crdList, err := clients.ApiExtensions.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Warn("Failed to list CRDs", "error", err)
		} else {
			for _, crd := range crdList.Items {
				key := crd.Spec.Group + "/" + crd.Spec.Names.Plural
				crdKeys[key] = struct{}{}
			}
		}
	}

	resourcesByKey := make(map[string]DiscoveryResource)
	for _, list := range resourceLists {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			log.Warn("Skipping invalid group version", "group_version", list.GroupVersion, "error", err)
			continue
		}

		for _, res := range list.APIResources {
			if strings.Contains(res.Name, "/") {
				continue
			}
			if !slices.Contains(res.Verbs, "list") {
				continue
			}

			key := gv.Group + "/" + res.Name
			_, isCRD := crdKeys[key]

			resourcesByKey[key] = DiscoveryResource{
				Group:        gv.Group,
				Version:      gv.Version,
				GroupVersion: list.GroupVersion,
				Resource:     res.Name,
				Kind:         res.Kind,
				ShortNames:   res.ShortNames,
				Namespaced:   res.Namespaced,
				Verbs:        res.Verbs,
				IsCRD:        isCRD,
			}
		}
	}

	resources := make([]DiscoveryResource, 0, len(resourcesByKey))
	for _, resource := range resourcesByKey {
		resources = append(resources, resource)
	}

	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Group != resources[j].Group {
			return resources[i].Group < resources[j].Group
		}
		if resources[i].Resource != resources[j].Resource {
			return resources[i].Resource < resources[j].Resource
		}
		return resources[i].Version < resources[j].Version
	})

	return resources, nil
}

func BuildCollectionsConfigFromDiscovery(resources []DiscoveryResource) (*CollectionsConfig, error) {
	// CRD resources themselves are skipped because they are not useful for our purpose, and including them would add a lot of noise.
	// The CRD list is still used to discover resources from CRDs to then collect.
	const includeCRDDefinitions = false
	const crdDefinitionGroup = "apiextensions.k8s.io"
	const crdDefinitionResource = "customresourcedefinitions"

	collections := make([]ResourceCollection, 0, len(resources))
	resourceCounts := make(map[string]int)
	for _, res := range resources {
		if !includeCRDDefinitions && res.Group == crdDefinitionGroup && res.Resource == crdDefinitionResource {
			continue
		}
		resourceCounts[res.Resource]++
	}

	seen := make(map[string]struct{})
	for _, res := range resources {
		if !includeCRDDefinitions && res.Group == crdDefinitionGroup && res.Resource == crdDefinitionResource {
			continue
		}
		name := res.Resource
		if resourceCounts[res.Resource] > 1 {
			group := res.Group
			if group == "" {
				group = "core"
			}
			group = strings.ReplaceAll(group, ".", "_")
			name = name + "_" + group
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}

		apiPath := res.Resource
		if res.GroupVersion != "" {
			apiPath = res.GroupVersion + "/" + res.Resource
		}
		collections = append(collections, ResourceCollection{
			Name:              name,
			ResourceType:      strings.ReplaceAll(name, "-", "_"),
			Kind:              res.Kind,
			ShortNames:        res.ShortNames,
			APIPath:           apiPath,
			APIVersion:        res.GroupVersion,
			APIGroup:          res.Group,
			Plural:            res.Resource,
			Namespaced:        res.Namespaced,
			ClusterScoped:     !res.Namespaced,
			Enabled:           true,
			FetchMode:         defaultFetchModeForResource(res),
			SupportedClusters: []utils.ClusterType{utils.ClusterTypeKubernetes, utils.ClusterTypeOpenShift},
			Custom:            res.IsCRD,
		})
	}

	cfg := &CollectionsConfig{
		Collections: collections,
	}

	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func defaultFetchModeForResource(res DiscoveryResource) FetchMode {
	if res.Kind != "" {
		gvk := schema.GroupVersion{Group: res.Group, Version: res.Version}.WithKind(res.Kind)
		if mode, ok := nodes.TypedFetchModeHint(gvk); ok {
			switch mode {
			case nodes.FetchModeHintMetadata:
				return FetchModeMetadata
			case nodes.FetchModeHintFull:
				return FetchModeFull
			}
		}
	}

	if res.IsCRD {
		return FetchModeMetadata
	}

	return FetchModeFull
}
