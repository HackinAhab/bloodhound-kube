package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"bloodhound-kube/internal/bloodhound"
	"bloodhound-kube/internal/collector"
)

type NodePropertyMapper struct{}

func (m *NodePropertyMapper) MapProperties(resource any) (map[string]any, error) {
	nodeData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal node resource: %w", err)
	}

	var node collector.Node
	if err := json.Unmarshal(nodeData, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node: %w", err)
	}

	properties := map[string]any{
		"hostname": node.Hostname,
		"fqdn":     node.Name,
		"domain":   "kubernetes.cluster",

		"internal_ip": node.InternalIP,
		"external_ip": node.ExternalIP,
		"pod_cidr":    node.PodCIDR,

		"kubelet_version":   node.KubeletVersion,
		"container_runtime": node.ContainerRuntime,
		"os_image":          node.OSImage,
		"kernel_version":    node.KernelVersion,
		"architecture":      node.Architecture,
		"operating_system":  node.OperatingSystem,
		"unschedulable":     node.Unschedulable,
		"created_at":        node.CreatedAt,

		"cpu_capacity":                  getResourceValue(node.Capacity.CPU),
		"memory_capacity":               getResourceValue(node.Capacity.Memory),
		"ephemeral_storage_capacity":    getResourceValue(node.Capacity.EphemeralStorage),
		"pods_capacity":                 getResourceValue(node.Capacity.Pods),
		"cpu_allocatable":               getResourceValue(node.Allocatable.CPU),
		"memory_allocatable":            getResourceValue(node.Allocatable.Memory),
		"ephemeral_storage_allocatable": getResourceValue(node.Allocatable.EphemeralStorage),
		"pods_allocatable":              getResourceValue(node.Allocatable.Pods),
	}

	if node.Labels != nil {
		isControlPlane := false
		isMaster := false
		nodeRole := "worker"

		if role, exists := node.Labels["kubernetes.io/role"]; exists {
			nodeRole = role
			properties["node_role"] = role
		}
		if _, exists := node.Labels["node-role.kubernetes.io/master"]; exists {
			isMaster = true
			isControlPlane = true
			nodeRole = "master"
		}
		if _, exists := node.Labels["node-role.kubernetes.io/control-plane"]; exists {
			isControlPlane = true
			nodeRole = "control-plane"
		}

		properties["is_master"] = isMaster
		properties["is_control_plane"] = isControlPlane
		properties["effective_role"] = nodeRole

		if instanceType, exists := node.Labels["node.kubernetes.io/instance-type"]; exists {
			properties["instance_type"] = instanceType
		}
		if provider, exists := node.Labels["kubernetes.io/cloud-provider"]; exists {
			properties["cloud_provider"] = provider
		}

		if zone, exists := node.Labels["topology.kubernetes.io/zone"]; exists {
			properties["availability_zone"] = zone
		}
		if region, exists := node.Labels["topology.kubernetes.io/region"]; exists {
			properties["region"] = region
		}

		if hostname, exists := node.Labels["kubernetes.io/hostname"]; exists {
			properties["label_hostname"] = hostname
		}
		if arch, exists := node.Labels["kubernetes.io/arch"]; exists {
			properties["label_architecture"] = arch
		}
		if os, exists := node.Labels["kubernetes.io/os"]; exists {
			properties["label_os"] = os
		}
	}

	if node.Annotations != nil {
		annotationCount := len(node.Annotations)
		properties["annotations_count"] = annotationCount

		securityAnnotations := make(map[string]any)

		for key, value := range node.Annotations {
			lowerKey := strings.ToLower(key)
			if strings.Contains(lowerKey, "network") ||
				strings.Contains(lowerKey, "flannel") ||
				strings.Contains(lowerKey, "calico") ||
				strings.Contains(lowerKey, "weave") ||
				strings.Contains(lowerKey, "cilium") {
				securityAnnotations[key] = value
			}
			if strings.Contains(lowerKey, "policy") ||
				strings.Contains(lowerKey, "security") ||
				strings.Contains(lowerKey, "selinux") ||
				strings.Contains(lowerKey, "apparmor") {
				securityAnnotations[key] = value
			}
		}

		if len(securityAnnotations) > 0 {
			properties["security_annotations"] = securityAnnotations
		}
	}

	if len(node.Taints) > 0 {
		var criticalTaints []string
		var hasNoSchedule bool
		var hasNoExecute bool

		for _, taint := range node.Taints {
			if strings.Contains(taint.Key, "master") ||
				strings.Contains(taint.Key, "control-plane") ||
				strings.Contains(taint.Key, "node-role") {
				criticalTaints = append(criticalTaints, taint.Key+":"+taint.Effect)
			}

			if taint.Effect == "NoSchedule" {
				hasNoSchedule = true
			}
			if taint.Effect == "NoExecute" {
				hasNoExecute = true
			}
		}

		if len(criticalTaints) > 0 {
			properties["security_taints"] = criticalTaints
		}
		properties["blocks_scheduling"] = hasNoSchedule
		properties["evicts_pods"] = hasNoExecute
	}
	var securityIssues []string

	if properties["is_control_plane"] == true {
		securityIssues = append(securityIssues, "control_plane_node")
	}

	if properties["external_ip"] != nil && properties["external_ip"] != "" {
		securityIssues = append(securityIssues, "externally_accessible")
	}

	if properties["has_problems"] == true {
		securityIssues = append(securityIssues, "operational_problems")
	}

	if properties["blocks_scheduling"] != true {
		securityIssues = append(securityIssues, "accepts_workloads")
	}

	if len(securityIssues) > 0 {
		properties["security_issues"] = securityIssues
	}

	return properties, nil
}

type NodeParser struct {
	config bloodhound.ResourceConfig
}

func NewNodeParser() *NodeParser {
	return &NodeParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "node",
			PrimaryKind:    "Node",
			SecondaryKinds: []string{"ComputeNode"},
			PropertyMapper: &NodePropertyMapper{},
		},
	}
}

func (p *NodeParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *NodeParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *NodeParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *NodeParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	nodeData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal node resource: %w", err)
	}

	var node collector.Node
	if err := json.Unmarshal(nodeData, &node); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node: %w", err)
	}

	bhNode, err := bloodhound.CreateNodeWithConfig(
		p.config,
		resource.Type,
		resource.Namespace,
		node.Name,
		resource.Resource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}

// getResourceValue safely extracts resource values
func getResourceValue(value string) string {
	if value == "" {
		return "0"
	}
	return value
}
