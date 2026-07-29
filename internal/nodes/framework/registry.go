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

type FetchModeHint string

const (
	FetchModeHintFull     FetchModeHint = "full"
	FetchModeHintMetadata FetchModeHint = "metadata"
)

var builders = map[string]Builder{}
var typedBuilders = map[string]TypedBuilder{}
var builderSources = map[string]string{}
var typedBuilderSources = map[string]string{}
var typedBuilderFetchModeHints = map[string]FetchModeHint{}
var registrationConflicts int

func RegisterKind(kind string, builder Builder) {
	source := callerSource(1)
	if existing, ok := builderSources[kind]; ok {
		registrationConflicts++
		registryLogger().Error("Duplicate node builder registration", "kind", kind, "existing", existing, "new", source)
		return
	}
	builders[kind] = builder
	builderSources[kind] = source
}

func RegisterTyped(gvk schema.GroupVersionKind, builder TypedBuilder) {
	registerTypedWithMode(gvk, builder, "")
}

func RegisterTypedWithFetchMode(gvk schema.GroupVersionKind, builder TypedBuilder, mode FetchModeHint) {
	registerTypedWithMode(gvk, builder, mode)
}

func registerTypedWithMode(gvk schema.GroupVersionKind, builder TypedBuilder, mode FetchModeHint) {
	key := GVKKey(gvk)
	source := callerSource(1)
	if existing, ok := typedBuilderSources[key]; ok {
		registrationConflicts++
		registryLogger().Error("Duplicate typed node builder registration", "gvk", key, "existing", existing, "new", source)
		return
	}
	typedBuilders[key] = builder
	typedBuilderSources[key] = source
	if mode != "" {
		typedBuilderFetchModeHints[key] = mode
	}
}

func Build(resource map[string]any) (BuildResult, bool) {
	kind, _ := resource["kind"].(string)
	if builder, ok := builders[kind]; ok {
		return builder(resource)
	}
	return BuildResult{}, false
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

func RegisterTypedFromMapWithFetchMode(gvk schema.GroupVersionKind, builder Builder, mode FetchModeHint) {
	RegisterTypedWithFetchMode(gvk, func(obj runtime.Object) (BuildResult, bool) {
		resource, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return BuildResult{}, false
		}
		return builder(resource)
	}, mode)
}

func LogRegistrationSummary() {
	registryLogger().Info("Node registration summary", "builders", len(builders), "typed_builders", len(typedBuilders), "conflicts", registrationConflicts)
}

func TypedFetchModeHint(gvk schema.GroupVersionKind) (FetchModeHint, bool) {
	mode, ok := typedBuilderFetchModeHints[GVKKey(gvk)]
	return mode, ok
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
