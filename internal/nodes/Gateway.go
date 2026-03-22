package nodes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

func init() {
	RegisterTyped(schema.GroupVersion{Group: gatewayv1.GroupVersion.Group, Version: gatewayv1.GroupVersion.Version}.WithKind("Gateway"), BuildGatewayNode)
	RegisterTyped(schema.GroupVersion{Group: gatewayv1beta1.GroupVersion.Group, Version: gatewayv1beta1.GroupVersion.Version}.WithKind("Gateway"), BuildGatewayNode)
}

type Gateway struct {
	GraphNodeBase
}

func BuildGatewayNode(obj runtime.Object) (BuildResult, bool) {
	switch typed := obj.(type) {
	case *gatewayv1.Gateway:
		return buildGatewayFromV1(typed)
	case *gatewayv1beta1.Gateway:
		converted := gatewayv1.Gateway(*typed)
		return buildGatewayFromV1(&converted)
	default:
		return BuildResult{}, false
	}
}

func buildGatewayFromV1(gateway *gatewayv1.Gateway) (BuildResult, bool) {
	if gateway == nil {
		return BuildResult{}, false
	}
	name := gateway.Name
	if name == "" {
		return BuildResult{}, false
	}
	namespace := gateway.Namespace
	labelsMap := StringMapToAnyMap(gateway.Labels)
	annotationsMap := StringMapToAnyMap(gateway.Annotations)

	properties := map[string]any{
		"name":             name,
		"namespace":        namespace,
		"labels":           MapToSortedList(labelsMap),
		"annotations":      MapToSortedList(annotationsMap),
		"gatewayClassName": string(gateway.Spec.GatewayClassName),
		"listeners":        summarizeGatewayListeners(gateway.Spec.Listeners),
		"addresses":        summarizeGatewayAddresses(gateway.Status.Addresses),
	}

	base := NewGraphNodeBase("Gateway", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Gateway{
			GraphNodeBase: base,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func summarizeGatewayListeners(listeners []gatewayv1.Listener) []string {
	if len(listeners) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(listeners))
	for _, listener := range listeners {
		entry := fmt.Sprintf("%s:%d/%s", listener.Name, listener.Port, listener.Protocol)
		if listener.Hostname != nil && *listener.Hostname != "" {
			entry = entry + " host=" + string(*listener.Hostname)
		}
		items = append(items, entry)
	}
	return items
}

func summarizeGatewayAddresses(addresses []gatewayv1.GatewayStatusAddress) []string {
	if len(addresses) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if address.Type != nil {
			items = append(items, string(*address.Type)+":"+address.Value)
			continue
		}
		items = append(items, address.Value)
	}
	return items
}
