package nodes

import (
	securityv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
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

var builders = map[string]Builder{}
var typedBuilders = map[string]TypedBuilder{}

func init() {
	RegisterTyped(corev1.SchemeGroupVersion.WithKind("ConfigMap"), BuildConfigMapNode)
	registerTypedFromMap(corev1.SchemeGroupVersion.WithKind("Namespace"), BuildNamespaceNode)
	RegisterTyped(corev1.SchemeGroupVersion.WithKind("Node"), BuildNodeNode)
	RegisterTyped(corev1.SchemeGroupVersion.WithKind("PersistentVolume"), BuildPVNode)
	RegisterTyped(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"), BuildPVCNode)
	RegisterTyped(corev1.SchemeGroupVersion.WithKind("Pod"), BuildPodNode)
	RegisterTyped(corev1.SchemeGroupVersion.WithKind("Secret"), BuildSecretNode)
	RegisterTyped(corev1.SchemeGroupVersion.WithKind("Service"), BuildServiceNode)
	RegisterTyped(corev1.SchemeGroupVersion.WithKind("ServiceAccount"), BuildServiceAccountNode)

	RegisterTyped(appsv1.SchemeGroupVersion.WithKind("DaemonSet"), BuildDaemonSetNode)
	RegisterTyped(appsv1.SchemeGroupVersion.WithKind("Deployment"), BuildDeploymentNode)
	RegisterTyped(appsv1.SchemeGroupVersion.WithKind("StatefulSet"), BuildStatefulSetNode)

	RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("ClusterRole"), BuildClusterRoleNode)
	RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"), BuildClusterRoleBindingNode)
	RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("Role"), BuildRoleNode)
	RegisterTyped(rbacv1.SchemeGroupVersion.WithKind("RoleBinding"), BuildRoleBindingNode)

	RegisterTyped(networkingv1.SchemeGroupVersion.WithKind("Ingress"), BuildIngressNode)
	RegisterTyped(networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy"), BuildNetworkPolicyNode)

	RegisterTyped(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("HTTPRoute"), BuildHTTPRouteNode)
	RegisterTyped(schema.GroupVersion{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version}.WithKind("HTTPRoute"), BuildHTTPRouteNode)
	RegisterTyped(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("GRPCRoute"), BuildGRPCRouteNode)
	RegisterTyped(schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("GRPCRoute"), BuildGRPCRouteNode)
	RegisterTyped(schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("TCPRoute"), BuildTCPRouteNode)
	RegisterTyped(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("TLSRoute"), BuildTLSRouteNode)
	RegisterTyped(schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}.WithKind("TLSRoute"), BuildTLSRouteNode)
	RegisterTyped(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("Gateway"), BuildGatewayNode)
	RegisterTyped(schema.GroupVersion{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version}.WithKind("Gateway"), BuildGatewayNode)

	RegisterTyped(securityv1.SchemeGroupVersion.WithKind("SecurityContextConstraints"), BuildSecurityContextConstraintsNode)
}

func Register(kind string, builder Builder) {
	builders[kind] = builder
}

func RegisterTyped(gvk schema.GroupVersionKind, builder TypedBuilder) {
	typedBuilders[GVKKey(gvk)] = builder
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

func GVKKey(gvk schema.GroupVersionKind) string {
	if gvk.Group == "" {
		return gvk.Version + "/" + gvk.Kind
	}
	return gvk.Group + "/" + gvk.Version + "/" + gvk.Kind
}

func registerTypedFromMap(gvk schema.GroupVersionKind, builder Builder) {
	RegisterTyped(gvk, func(obj runtime.Object) (BuildResult, bool) {
		resource, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return BuildResult{}, false
		}
		return builder(resource)
	})
}
