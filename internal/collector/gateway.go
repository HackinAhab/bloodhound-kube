package collector

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Gateway struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Class       string            `json:"class,omitempty"`
	Listeners   []GatewayListener `json:"listeners,omitempty"`
	Status      GatewayStatus     `json:"status,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

type GatewayListener struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
	Hostname string `json:"hostname,omitempty"`
}

type GatewayStatus struct {
	Conditions []GatewayCondition `json:"conditions,omitempty"`
	Addresses  []GatewayAddress   `json:"addresses,omitempty"`
}

type GatewayCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type GatewayAddress struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value"`
}

func (c *Collector) CollectGateways(ctx context.Context, namespace string) ([]Gateway, error) {
	c.logger.Info("Collecting gateways", "namespace", namespace)

	gatewayGVR := schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "gateways",
	}

	dynamicClient := c.client.Discovery().RESTClient()
	result := dynamicClient.Get().
		AbsPath("/apis", gatewayGVR.Group, gatewayGVR.Version, "namespaces", namespace, gatewayGVR.Resource).
		Do(ctx)

	rawData, err := result.Raw()
	if err != nil {
		c.logger.Debug("Gateway API not available or no gateways found", "error", err)
		return []Gateway{}, nil
	}

	gatewayList := &unstructured.UnstructuredList{}
	if err := gatewayList.UnmarshalJSON(rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gateway list: %w", err)
	}

	gateways := make([]Gateway, 0, len(gatewayList.Items))
	for _, item := range gatewayList.Items {
		gateway := Gateway{
			Name:        item.GetName(),
			Namespace:   item.GetNamespace(),
			Labels:      item.GetLabels(),
			Annotations: item.GetAnnotations(),
		}

		if creationTime := item.GetCreationTimestamp(); !creationTime.IsZero() {
			gateway.CreatedAt = creationTime.Format("2006-01-02T15:04:05Z")
		}

		if spec, found, _ := unstructured.NestedMap(item.Object, "spec"); found {
			if gatewayClassName, found, _ := unstructured.NestedString(spec, "gatewayClassName"); found {
				gateway.Class = gatewayClassName
			}

			if listenersRaw, found, _ := unstructured.NestedSlice(spec, "listeners"); found {
				var listeners []GatewayListener
				for _, listenerRaw := range listenersRaw {
					if listenerMap, ok := listenerRaw.(map[string]any); ok {
						listener := GatewayListener{}
						if name, found, _ := unstructured.NestedString(listenerMap, "name"); found {
							listener.Name = name
						}
						if port, found, _ := unstructured.NestedInt64(listenerMap, "port"); found {
							listener.Port = int32(port)
						}
						if protocol, found, _ := unstructured.NestedString(listenerMap, "protocol"); found {
							listener.Protocol = protocol
						}
						if hostname, found, _ := unstructured.NestedString(listenerMap, "hostname"); found {
							listener.Hostname = hostname
						}
						listeners = append(listeners, listener)
					}
				}
				gateway.Listeners = listeners
			}
		}

		if status, found, _ := unstructured.NestedMap(item.Object, "status"); found {
			gatewayStatus := GatewayStatus{}

			if conditionsRaw, found, _ := unstructured.NestedSlice(status, "conditions"); found {
				var conditions []GatewayCondition
				for _, conditionRaw := range conditionsRaw {
					if conditionMap, ok := conditionRaw.(map[string]any); ok {
						condition := GatewayCondition{}
						if condType, found, _ := unstructured.NestedString(conditionMap, "type"); found {
							condition.Type = condType
						}
						if condStatus, found, _ := unstructured.NestedString(conditionMap, "status"); found {
							condition.Status = condStatus
						}
						if reason, found, _ := unstructured.NestedString(conditionMap, "reason"); found {
							condition.Reason = reason
						}
						conditions = append(conditions, condition)
					}
				}
				gatewayStatus.Conditions = conditions
			}

			if addressesRaw, found, _ := unstructured.NestedSlice(status, "addresses"); found {
				var addresses []GatewayAddress
				for _, addressRaw := range addressesRaw {
					if addressMap, ok := addressRaw.(map[string]any); ok {
						address := GatewayAddress{}
						if addrType, found, _ := unstructured.NestedString(addressMap, "type"); found {
							address.Type = addrType
						}
						if value, found, _ := unstructured.NestedString(addressMap, "value"); found {
							address.Value = value
						}
						addresses = append(addresses, address)
					}
				}
				gatewayStatus.Addresses = addresses
			}

			gateway.Status = gatewayStatus
		}

		gateways = append(gateways, gateway)
	}

	c.logger.Info("Successfully collected gateways", "count", len(gateways))
	return gateways, nil
}

type GatewaysHandler struct {
	*BaseHandler
}

func NewGatewaysHandler() *GatewaysHandler {
	return &GatewaysHandler{
		BaseHandler: &BaseHandler{
			name:          "gateways",
			clusterScoped: false,
		},
	}
}

func (h *GatewaysHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	gateways, err := c.CollectGateways(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(gateways))

	for _, gateway := range gateways {
		batch = append(batch, Resource{
			Type:      "gateway",
			Namespace: namespace,
			Resource:  gateway,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
