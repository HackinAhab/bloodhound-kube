package collector

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Node struct {
	Name             string            `json:"name"`
	Labels           map[string]string `json:"labels,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	CreatedAt        string            `json:"created_at"`
	Ready            bool              `json:"ready"`
	KubeletVersion   string            `json:"kubelet_version"`
	ContainerRuntime string            `json:"container_runtime"`
	OSImage          string            `json:"os_image"`
	KernelVersion    string            `json:"kernel_version"`
	Architecture     string            `json:"architecture"`
	OperatingSystem  string            `json:"operating_system"`
	InternalIP       string            `json:"internal_ip,omitempty"`
	ExternalIP       string            `json:"external_ip,omitempty"`
	Hostname         string            `json:"hostname,omitempty"`
	PodCIDR          string            `json:"pod_cidr,omitempty"`
	Unschedulable    bool              `json:"unschedulable"`
	Taints           []NodeTaint       `json:"taints,omitempty"`
	Conditions       []NodeCondition   `json:"conditions,omitempty"`
	Capacity         ResourceList      `json:"capacity,omitempty"`
	Allocatable      ResourceList      `json:"allocatable,omitempty"`
}

type NodeTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}

type NodeCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type ResourceList struct {
	CPU              string `json:"cpu,omitempty"`
	Memory           string `json:"memory,omitempty"`
	EphemeralStorage string `json:"ephemeral_storage,omitempty"`
	Pods             string `json:"pods,omitempty"`
}

func (c *Collector) CollectNodes(ctx context.Context) ([]Node, error) {
	c.logger.Info("Collecting nodes")

	nodeList, err := c.clients.Kubernetes.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodes := make([]Node, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		var taints []NodeTaint
		for _, taint := range node.Spec.Taints {
			taints = append(taints, NodeTaint{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: string(taint.Effect),
			})
		}

		var conditions []NodeCondition
		for _, condition := range node.Status.Conditions {
			conditions = append(conditions, NodeCondition{
				Type:    string(condition.Type),
				Status:  string(condition.Status),
				Reason:  condition.Reason,
				Message: condition.Message,
			})
		}

		ready := false
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				ready = condition.Status == corev1.ConditionTrue
				break
			}
		}

		var internalIP, externalIP, hostname string
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				internalIP = addr.Address
			case corev1.NodeExternalIP:
				externalIP = addr.Address
			case corev1.NodeHostName:
				hostname = addr.Address
			}
		}

		capacity := ResourceList{
			CPU:              node.Status.Capacity.Cpu().String(),
			Memory:           node.Status.Capacity.Memory().String(),
			EphemeralStorage: node.Status.Capacity.StorageEphemeral().String(),
			Pods:             node.Status.Capacity.Pods().String(),
		}

		allocatable := ResourceList{
			CPU:              node.Status.Allocatable.Cpu().String(),
			Memory:           node.Status.Allocatable.Memory().String(),
			EphemeralStorage: node.Status.Allocatable.StorageEphemeral().String(),
			Pods:             node.Status.Allocatable.Pods().String(),
		}

		nodes = append(nodes, Node{
			Name:             node.Name,
			Labels:           node.Labels,
			Annotations:      node.Annotations,
			CreatedAt:        node.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			Ready:            ready,
			KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
			ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			OSImage:          node.Status.NodeInfo.OSImage,
			KernelVersion:    node.Status.NodeInfo.KernelVersion,
			Architecture:     node.Status.NodeInfo.Architecture,
			OperatingSystem:  node.Status.NodeInfo.OperatingSystem,
			InternalIP:       internalIP,
			ExternalIP:       externalIP,
			Hostname:         hostname,
			PodCIDR:          node.Spec.PodCIDR,
			Unschedulable:    node.Spec.Unschedulable,
			Taints:           taints,
			Conditions:       conditions,
			Capacity:         capacity,
			Allocatable:      allocatable,
		})
	}

	c.logger.Info("Successfully collected nodes", "count", len(nodes))
	return nodes, nil
}

type NodesHandler struct {
	*BaseHandler
}

func NewNodesHandler() *NodesHandler {
	return &NodesHandler{
		BaseHandler: &BaseHandler{
			name:          "nodes",
			clusterScoped: true,
		},
	}
}

func (h *NodesHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	nodes, err := c.CollectNodes(ctx)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(nodes))

	for _, node := range nodes {
		batch = append(batch, Resource{
			Type:      "node",
			Resource:  node,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
