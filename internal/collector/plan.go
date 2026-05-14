package collector

import (
	"fmt"
	"sort"
	"strings"

	"bloodhound-kube/internal/utils"
)

type CollectionTarget struct {
	Name          string
	Kind          string
	ShortNames    []string
	APIPath       string
	Group         string
	Version       string
	GroupVersion  string
	Resource      string
	Namespaced    bool
	ClusterScoped bool
	FetchMode     FetchMode
}

type CollectionPlan struct {
	Targets     []CollectionTarget
	lookup      map[string]string
	ambiguous   map[string]struct{}
	targetByKey map[string]CollectionTarget
}

func BuildCollectionPlan(resources []DiscoveryResource) (*CollectionPlan, error) {
	collectionsCfg, err := BuildCollectionsConfigFromDiscovery(resources)
	if err != nil {
		return nil, err
	}
	return NewCollectionPlan(collectionsCfg), nil
}

func NewCollectionPlan(cfg *CollectionsConfig) *CollectionPlan {
	plan := &CollectionPlan{
		Targets:     make([]CollectionTarget, 0, len(cfg.Collections)),
		lookup:      make(map[string]string),
		ambiguous:   make(map[string]struct{}),
		targetByKey: make(map[string]CollectionTarget),
	}

	addAlias := func(alias, canonical string) {
		key := normalizeTypeKey(alias)
		if key == "" {
			return
		}
		if existing, ok := plan.lookup[key]; ok {
			if existing != canonical {
				delete(plan.lookup, key)
				plan.ambiguous[key] = struct{}{}
			}
			return
		}
		if _, ok := plan.ambiguous[key]; ok {
			return
		}
		plan.lookup[key] = canonical
	}

	for _, collection := range cfg.Collections {
		if !collection.Enabled {
			continue
		}
		if !collection.SupportsCluster(utils.ClusterTypeKubernetes) && !collection.SupportsCluster(utils.ClusterTypeOpenShift) {
			continue
		}
		target := CollectionTarget{
			Name:          collection.Name,
			Kind:          collection.Kind,
			ShortNames:    collection.ShortNames,
			APIPath:       collection.APIPath,
			GroupVersion:  collection.APIVersion,
			Group:         collection.APIGroup,
			Version:       extractVersion(collection.APIVersion),
			Resource:      collection.Plural,
			Namespaced:    collection.Namespaced,
			ClusterScoped: collection.ClusterScoped,
			FetchMode:     collection.FetchMode,
		}
		plan.Targets = append(plan.Targets, target)
		plan.targetByKey[collection.Name] = target

		addAlias(collection.Name, collection.Name)
		addAlias(collection.Kind, collection.Name)
		for _, shortName := range collection.ShortNames {
			addAlias(shortName, collection.Name)
		}
		addAlias(collection.APIPath, collection.Name)
	}

	sort.Slice(plan.Targets, func(i, j int) bool {
		return plan.Targets[i].Name < plan.Targets[j].Name
	})

	return plan
}

func (p *CollectionPlan) Names() []string {
	names := make([]string, 0, len(p.Targets))
	for _, target := range p.Targets {
		names = append(names, target.Name)
	}
	return names
}

func (p *CollectionPlan) NormalizeTypes(types []string) ([]string, error) {
	if len(types) == 0 {
		return nil, nil
	}
	resolved := make([]string, 0, len(types))
	var invalid []string
	for _, resourceType := range types {
		key := normalizeTypeKey(resourceType)
		if key == "" {
			invalid = append(invalid, resourceType)
			continue
		}
		if _, exists := p.ambiguous[key]; exists {
			invalid = append(invalid, resourceType)
			continue
		}
		canonical, ok := p.lookup[key]
		if !ok {
			invalid = append(invalid, resourceType)
			continue
		}
		resolved = append(resolved, canonical)
	}
	if len(invalid) > 0 {
		return nil, fmt.Errorf("unsupported resource types: %s (available: %s)", strings.Join(invalid, ", "), strings.Join(p.Names(), ", "))
	}
	return resolved, nil
}

func (p *CollectionPlan) TargetsForTypes(types []string) ([]CollectionTarget, error) {
	if len(types) == 0 {
		return p.Targets, nil
	}
	normalized, err := p.NormalizeTypes(types)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(normalized))
	selected := make([]CollectionTarget, 0, len(normalized))
	for _, name := range normalized {
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		target, ok := p.targetByKey[name]
		if !ok {
			continue
		}
		selected = append(selected, target)
	}
	return selected, nil
}
