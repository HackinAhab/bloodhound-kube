package nodes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Service struct {
	GraphNodeBase
	Ports       []any
	ServiceType string
}

func BuildServiceNode(obj runtime.Object) (BuildResult, bool) {
	svc, ok := obj.(*corev1.Service)
	if !ok || svc == nil {
		return BuildResult{}, false
	}
	name := svc.Name
	if name == "" {
		return BuildResult{}, false
	}

	namespace := svc.Namespace
	labelsMap := StringMapToAnyMap(svc.Labels)
	annotationsMap := StringMapToAnyMap(svc.Annotations)
	selectorMap := StringMapToAnyMap(svc.Spec.Selector)

	ports := servicePortsToAnySlice(svc.Spec.Ports)
	serviceType := string(svc.Spec.Type)

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"selector":    MapToSortedList(selectorMap),
		"serviceType": serviceType,
		"clusterIP":   svc.Spec.ClusterIP,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: Service{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("Service", namespace, name),
				Kinds:          []string{"Service"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			Ports:       ports,
			ServiceType: serviceType,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("Service", namespace, name),
			Kinds:      []string{"Service"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func servicePortsToAnySlice(ports []corev1.ServicePort) []any {
	if len(ports) == 0 {
		return []any{}
	}
	items := make([]any, 0, len(ports))
	for _, port := range ports {
		targetPort := ""
		if port.TargetPort.Type != 0 || port.TargetPort.IntVal != 0 || port.TargetPort.StrVal != "" {
			targetPort = port.TargetPort.String()
		}
		entry := map[string]any{
			"name":     port.Name,
			"protocol": string(port.Protocol),
			"port":     port.Port,
		}
		if targetPort != "" {
			entry["targetPort"] = targetPort
		}
		if port.NodePort != 0 {
			entry["nodePort"] = port.NodePort
		}
		if port.AppProtocol != nil && *port.AppProtocol != "" {
			entry["appProtocol"] = *port.AppProtocol
		}
		items = append(items, entry)
	}
	return items
}
