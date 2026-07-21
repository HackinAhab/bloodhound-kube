package nodes

import (
	"sync"

	"bloodhound-kube/internal/nodes/addons"
	"bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/mounts"
	"bloodhound-kube/internal/nodes/networking"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/nodes/workload"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type NodeResult = framework.NodeResult
type CoreEntry = framework.CoreEntry
type BuildResult = framework.BuildResult
type Builder = framework.Builder
type TypedBuilder = framework.TypedBuilder
type FetchModeHint = framework.FetchModeHint

const (
	FetchModeHintFull     = framework.FetchModeHintFull
	FetchModeHintMetadata = framework.FetchModeHintMetadata
)

var registerOnce sync.Once

func init() {
	ensureRegistered()
}

func ensureRegistered() {
	registerOnce.Do(func() {
		rbac.Register()
		networking.Register()
		workload.Register()
		mounts.Register()
		addons.Register()
		platform.Register()

		framework.LogRegistrationSummary()
	})
}

func Register(kind string, builder Builder) {
	framework.RegisterKind(kind, builder)
}

func RegisterTyped(gvk schema.GroupVersionKind, builder TypedBuilder) {
	framework.RegisterTyped(gvk, builder)
}

func RegisterTypedWithFetchMode(gvk schema.GroupVersionKind, builder TypedBuilder, mode FetchModeHint) {
	framework.RegisterTypedWithFetchMode(gvk, builder, mode)
}

func RegisterTypedFromMapWithFetchMode(gvk schema.GroupVersionKind, builder Builder, mode FetchModeHint) {
	framework.RegisterTypedFromMapWithFetchMode(gvk, builder, mode)
}

func Build(resource map[string]any) (BuildResult, bool) {
	ensureRegistered()
	return framework.Build(resource)
}

func BuildTyped(gvk schema.GroupVersionKind, obj runtime.Object) (BuildResult, bool) {
	ensureRegistered()
	return framework.BuildTyped(gvk, obj)
}

func BuildTypedFromMap(gvk schema.GroupVersionKind, resource map[string]any) (BuildResult, bool) {
	ensureRegistered()
	return framework.BuildTypedFromMap(gvk, resource)
}

func GVKKey(gvk schema.GroupVersionKind) string {
	return framework.GVKKey(gvk)
}

func TypedFetchModeHint(gvk schema.GroupVersionKind) (FetchModeHint, bool) {
	ensureRegistered()
	return framework.TypedFetchModeHint(gvk)
}
