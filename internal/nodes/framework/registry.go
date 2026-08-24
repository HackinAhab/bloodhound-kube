package framework

import (
	"fmt"
	"path/filepath"
	goruntime "runtime"

	"bloodhound-kube/internal/utils"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type NodeResult struct {
	ID         string
	Kinds      []string
	Properties map[string]any
}

type CoreEntry struct {
	Namespace string
	Cluster   bool
	Data      any
}

type BuildResult struct {
	Node NodeResult
	Core []CoreEntry
}

type Builder func(resource map[string]any) (BuildResult, bool)

type TypedBuilder func(obj runtime.Object) (BuildResult, bool)

// ResourceRegistration describes one supported Kubernetes resource for parser
// dispatch and fetch policy.
type ResourceRegistration struct {
	GVK          schema.GroupVersionKind
	TypedBuilder TypedBuilder
	MapBuilder   Builder
	FetchMode    FetchModeHint
}

// CollectionTarget declares a resource selected by the default collection
// scope. An empty Resource selects every resource in the group/version.
type CollectionTarget struct {
	Group    string
	Version  string
	Resource string
}

type FetchModeHint string

const (
	FetchModeHintFull     FetchModeHint = "full"
	FetchModeHintMetadata FetchModeHint = "metadata"
)

var typedBuilders = map[string]TypedBuilder{}
var typedBuilderSources = map[string]string{}
var typedBuilderFetchModeHints = map[string]FetchModeHint{}
var defaultCollectionTargets []CollectionTarget
var registrationConflicts int

func RegisterResources(resources ...ResourceRegistration) {
	for _, resource := range resources {
		registerResource(resource)
	}
}

func RegisterDefaultCollections(targets ...CollectionTarget) {
	defaultCollectionTargets = append(defaultCollectionTargets, targets...)
}

func registerResource(resource ResourceRegistration) {
	if resource.GVK.Kind == "" {
		registrationConflicts++
		registryLogger().Error("Invalid node resource registration", "reason", "missing kind")
		return
	}
	if (resource.TypedBuilder == nil) == (resource.MapBuilder == nil) {
		registrationConflicts++
		registryLogger().Error("Invalid node resource registration", "gvk", GVKKey(resource.GVK), "reason", "exactly one builder is required")
		return
	}
	builder := resource.TypedBuilder
	if resource.MapBuilder != nil {
		builder = func(obj runtime.Object) (BuildResult, bool) {
			mapped, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
			if err != nil {
				return BuildResult{}, false
			}
			return resource.MapBuilder(mapped)
		}
	}

	key := GVKKey(resource.GVK)
	source := callerSource(2)
	if existing, ok := typedBuilderSources[key]; ok {
		registrationConflicts++
		registryLogger().Error("Duplicate node resource registration", "gvk", key, "existing", existing, "new", source)
		return
	}
	typedBuilders[key] = builder
	typedBuilderSources[key] = source
	if resource.FetchMode != "" {
		typedBuilderFetchModeHints[key] = resource.FetchMode
	}
}

func BuildTyped(gvk schema.GroupVersionKind, obj runtime.Object) (BuildResult, bool) {
	if builder, ok := typedBuilders[GVKKey(gvk)]; ok {
		return builder(obj)
	}
	return BuildResult{}, false
}

func BuildTypedFromMap(gvk schema.GroupVersionKind, resource map[string]any) (BuildResult, bool) {
	builder, ok := typedBuilders[GVKKey(gvk)]
	if !ok {
		return BuildResult{}, false
	}
	obj := &unstructured.Unstructured{Object: resource}
	return builder(obj)
}

func GVKKey(gvk schema.GroupVersionKind) string {
	if gvk.Group == "" {
		return gvk.Version + "/" + gvk.Kind
	}
	return gvk.Group + "/" + gvk.Version + "/" + gvk.Kind
}

func LogRegistrationSummary() {
	registryLogger().Info("Node registration summary", "resources", len(typedBuilders), "conflicts", registrationConflicts)
}

func TypedFetchModeHint(gvk schema.GroupVersionKind) (FetchModeHint, bool) {
	mode, ok := typedBuilderFetchModeHints[GVKKey(gvk)]
	return mode, ok
}

func DefaultCollectionTargets() []CollectionTarget {
	targets := make([]CollectionTarget, len(defaultCollectionTargets))
	copy(targets, defaultCollectionTargets)
	return targets
}

func callerSource(skip int) string {
	pc, file, line, ok := goruntime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	fn := goruntime.FuncForPC(pc)
	fnName := "unknown"
	if fn != nil {
		fnName = fn.Name()
	}
	return fmt.Sprintf("%s:%d (%s)", filepath.Base(file), line, fnName)
}

func registryLogger() *utils.Logger {
	return utils.DefaultLogger().Component("nodes.registry")
}
