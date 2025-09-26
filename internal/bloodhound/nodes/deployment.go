package nodes

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"bloodhound-kube/internal/bloodhound"
)

type DeploymentPropertyMapper struct{}

func (m *DeploymentPropertyMapper) MapProperties(resource any) (map[string]any, error) {
	// Try to cast to structured collector.Deployment first
	if deployment, ok := resource.(map[string]any); ok {
		// This is structured data from the collector
		return m.mapFromStructuredData(deployment)
	}

	// Fallback to the original approach for backwards compatibility
	deploymentData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment resource: %w", err)
	}

	var deployment map[string]any
	if err := json.Unmarshal(deploymentData, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	return m.mapFromStructuredData(deployment)
}

func (m *DeploymentPropertyMapper) mapFromStructuredData(deployment map[string]any) (map[string]any, error) {
	properties := map[string]any{}

	// Handle structured collector format - check if this is from collector
	if m.isCollectorFormat(deployment) {
		return m.mapFromCollectorFormat(deployment, properties)
	}

	// Handle legacy format
	spec, _ := deployment["spec"].(map[string]any)
	if spec == nil {
		return properties, nil
	}

	// Basic deployment properties
	if replicas, ok := spec["replicas"]; ok {
		properties["spec_replicas"] = replicas
	}

	if strategy, ok := spec["strategy"].(map[string]any); ok {
		if strategyType, ok := strategy["type"].(string); ok {
			properties["spec_strategy_type"] = strategyType
			properties["is_rolling_update"] = strategyType == "RollingUpdate"
			properties["is_recreate"] = strategyType == "Recreate"
		}

		if rollingUpdate, ok := strategy["rollingUpdate"].(map[string]any); ok {
			if maxSurge, ok := rollingUpdate["maxSurge"]; ok {
				properties["spec_strategy_rolling_update_max_surge"] = maxSurge
			}
			if maxUnavailable, ok := rollingUpdate["maxUnavailable"]; ok {
				properties["spec_strategy_rolling_update_max_unavailable"] = maxUnavailable
			}
		}
	}

	if revisionHistoryLimit, ok := spec["revisionHistoryLimit"]; ok {
		properties["spec_revision_history_limit"] = revisionHistoryLimit
	}

	if progressDeadlineSeconds, ok := spec["progressDeadlineSeconds"]; ok {
		properties["spec_progress_deadline_seconds"] = progressDeadlineSeconds
	}

	// Template spec properties
	if template, ok := spec["template"].(map[string]any); ok {
		if templateSpec, ok := template["spec"].(map[string]any); ok {
			m.mapPodTemplateSpec(templateSpec, properties)
		}
	}

	// Selector
	if selector, ok := spec["selector"].(map[string]any); ok {
		if matchLabels, ok := selector["matchLabels"].(map[string]any); ok {
			properties["has_selector"] = len(matchLabels) > 0
			properties["selector_count"] = len(matchLabels)

			var selectors []string
			for key, value := range matchLabels {
				if valueStr, ok := value.(string); ok {
					selectors = append(selectors, fmt.Sprintf("%s=%s", key, valueStr))
				}
			}
			if len(selectors) > 0 {
				properties["spec_selector_match_labels"] = strings.Join(selectors, ",")
			}

			// Common selector patterns
			if app, exists := matchLabels["app"]; exists {
				properties["selects_by_app"] = true
				properties["app_selector"] = app
			}
		}
	}

	// Status information
	if status, ok := deployment["status"].(map[string]any); ok {
		m.mapStatus(status, properties)
	}

	return properties, nil
}

func (m *DeploymentPropertyMapper) mapPodTemplateSpec(templateSpec map[string]any, properties map[string]any) {
	if restartPolicy, ok := templateSpec["restartPolicy"].(string); ok {
		properties["spec_template_restart_policy"] = restartPolicy
	}

	if dnsPolicy, ok := templateSpec["dnsPolicy"].(string); ok {
		properties["spec_template_dns_policy"] = dnsPolicy
	}

	if schedulerName, ok := templateSpec["schedulerName"].(string); ok {
		properties["spec_template_scheduler_name"] = schedulerName
	}

	if serviceAccountName, ok := templateSpec["serviceAccountName"].(string); ok {
		properties["spec_template_service_account_name"] = serviceAccountName
		properties["uses_service_account"] = serviceAccountName != ""
	}

	// Security context
	if securityContext, ok := templateSpec["securityContext"].(map[string]any); ok {
		m.mapSecurityContext(securityContext, properties)
	}

	// Containers
	if containers, ok := templateSpec["containers"].([]any); ok {
		m.mapContainers(containers, properties)
	}

	// Node selector
	if nodeSelector, ok := templateSpec["nodeSelector"].(map[string]any); ok && len(nodeSelector) > 0 {
		properties["has_node_selector"] = true
		properties["node_selector_count"] = len(nodeSelector)

		var selectors []string
		for key, value := range nodeSelector {
			if valueStr, ok := value.(string); ok {
				selectors = append(selectors, fmt.Sprintf("%s=%s", key, valueStr))
			}
		}
		properties["spec_template_node_selector"] = strings.Join(selectors, ",")
	}

	// Tolerations
	if tolerations, ok := templateSpec["tolerations"].([]any); ok && len(tolerations) > 0 {
		properties["has_tolerations"] = true
		properties["tolerations_count"] = len(tolerations)

		var tolerationStrs []string
		for _, toleration := range tolerations {
			if tolerationMap, ok := toleration.(map[string]any); ok {
				key, _ := tolerationMap["key"].(string)
				operator, _ := tolerationMap["operator"].(string)
				effect, _ := tolerationMap["effect"].(string)
				tolerationStrs = append(tolerationStrs, fmt.Sprintf("%s:%s:%s", key, operator, effect))
			}
		}
		properties["spec_template_tolerations"] = strings.Join(tolerationStrs, ",")
	}

	// Affinity
	if affinity, ok := templateSpec["affinity"]; ok && affinity != nil {
		properties["spec_template_has_affinity"] = true
	} else {
		properties["spec_template_has_affinity"] = false
	}
}

func (m *DeploymentPropertyMapper) mapSecurityContext(securityContext map[string]any, properties map[string]any) {
	if runAsUser, ok := securityContext["runAsUser"]; ok {
		properties["spec_template_security_context_run_as_user"] = runAsUser
	}

	if runAsGroup, ok := securityContext["runAsGroup"]; ok {
		properties["spec_template_security_context_run_as_group"] = runAsGroup
	}

	if fsGroup, ok := securityContext["fsGroup"]; ok {
		properties["spec_template_security_context_fs_group"] = fsGroup
	}

	if runAsNonRoot, ok := securityContext["runAsNonRoot"].(bool); ok {
		properties["spec_template_security_context_run_as_non_root"] = runAsNonRoot
	}
}

func (m *DeploymentPropertyMapper) mapContainers(containers []any, properties map[string]any) {
	properties["container_count"] = len(containers)

	var containerNames, images []string
	var ports []int32
	var envVars []string
	hasPrivilegedPorts := false
	hasResourceLimits := false
	hasResourceRequests := false
	hasLivenessProbe := false
	hasReadinessProbe := false

	for _, container := range containers {
		if containerMap, ok := container.(map[string]any); ok {
			// Container name and image
			if name, ok := containerMap["name"].(string); ok {
				containerNames = append(containerNames, name)
			}
			if image, ok := containerMap["image"].(string); ok {
				images = append(images, image)
			}

			// Ports
			if containerPorts, ok := containerMap["ports"].([]any); ok {
				for _, port := range containerPorts {
					if portMap, ok := port.(map[string]any); ok {
						if containerPort, ok := portMap["containerPort"]; ok {
							if portInt, err := strconv.Atoi(fmt.Sprintf("%.0f", containerPort)); err == nil {
								ports = append(ports, int32(portInt))
								if portInt < 1024 {
									hasPrivilegedPorts = true
								}
							}
						}
					}
				}
			}

			// Environment variables
			if env, ok := containerMap["env"].([]any); ok {
				for _, envVar := range env {
					if envMap, ok := envVar.(map[string]any); ok {
						name, _ := envMap["name"].(string)
						value, _ := envMap["value"].(string)
						if name != "" {
							envVars = append(envVars, fmt.Sprintf("%s=%s", name, value))
						}
					}
				}
			}

			// Resources
			if resources, ok := containerMap["resources"].(map[string]any); ok {
				if limits, ok := resources["limits"]; ok && limits != nil {
					hasResourceLimits = true
				}
				if requests, ok := resources["requests"]; ok && requests != nil {
					hasResourceRequests = true
				}
			}

			// Health checks
			if _, ok := containerMap["livenessProbe"]; ok {
				hasLivenessProbe = true
			}
			if _, ok := containerMap["readinessProbe"]; ok {
				hasReadinessProbe = true
			}
		}
	}

	if len(containerNames) > 0 {
		properties["spec_template_containers"] = strings.Join(containerNames, ",")
	}
	if len(images) > 0 {
		properties["spec_template_images"] = strings.Join(images, ",")
	}
	if len(ports) > 0 {
		properties["spec_template_ports"] = ports
	}
	if len(envVars) > 0 {
		properties["spec_template_env_vars"] = strings.Join(envVars, ",")
	}

	properties["has_privileged_ports"] = hasPrivilegedPorts
	properties["has_resource_limits"] = hasResourceLimits
	properties["has_resource_requests"] = hasResourceRequests
	properties["has_liveness_probe"] = hasLivenessProbe
	properties["has_readiness_probe"] = hasReadinessProbe
	properties["has_health_checks"] = hasLivenessProbe || hasReadinessProbe
}

func (m *DeploymentPropertyMapper) mapStatus(status map[string]any, properties map[string]any) {
	if replicas, ok := status["replicas"]; ok {
		properties["status_replicas"] = replicas
	}

	if updatedReplicas, ok := status["updatedReplicas"]; ok {
		properties["status_updated_replicas"] = updatedReplicas
	}

	if readyReplicas, ok := status["readyReplicas"]; ok {
		properties["status_ready_replicas"] = readyReplicas
	}

	if availableReplicas, ok := status["availableReplicas"]; ok {
		properties["status_available_replicas"] = availableReplicas
	}

	if unavailableReplicas, ok := status["unavailableReplicas"]; ok {
		properties["status_unavailable_replicas"] = unavailableReplicas
	}

	if observedGeneration, ok := status["observedGeneration"]; ok {
		properties["status_observed_generation"] = observedGeneration
	}

	// Conditions
	if conditions, ok := status["conditions"].([]any); ok && len(conditions) > 0 {
		properties["has_conditions"] = true
		properties["condition_count"] = len(conditions)

		var conditionStrs []string
		var isProgressing, isAvailable bool

		for _, condition := range conditions {
			if conditionMap, ok := condition.(map[string]any); ok {
				conditionType, _ := conditionMap["type"].(string)
				conditionStatus, _ := conditionMap["status"].(string)

				conditionStrs = append(conditionStrs, fmt.Sprintf("%s:%s", conditionType, conditionStatus))

				if conditionType == "Progressing" && conditionStatus == "True" {
					isProgressing = true
				}
				if conditionType == "Available" && conditionStatus == "True" {
					isAvailable = true
				}
			}
		}

		properties["status_conditions"] = strings.Join(conditionStrs, ",")
		properties["is_progressing"] = isProgressing
		properties["is_available"] = isAvailable
	}
}

// isCollectorFormat checks if the deployment data is from the structured collector
func (m *DeploymentPropertyMapper) isCollectorFormat(deployment map[string]any) bool {
	// Check for collector-specific fields that indicate structured format
	if _, hasSpec := deployment["spec"]; hasSpec {
		if _, hasStatus := deployment["status"]; hasStatus {
			// This looks like the structured collector format
			if spec, ok := deployment["spec"].(map[string]any); ok {
				// Check for specific collector format fields
				if _, hasReplicas := spec["replicas"]; hasReplicas {
					return true
				}
				if _, hasSelector := spec["selector"]; hasSelector {
					return true
				}
			}
		}
	}
	return false
}

// mapFromCollectorFormat maps from the structured collector format
func (m *DeploymentPropertyMapper) mapFromCollectorFormat(deployment map[string]any, properties map[string]any) (map[string]any, error) {
	// Extract basic metadata
	if name, ok := deployment["name"].(string); ok {
		properties["name"] = name
	}
	if namespace, ok := deployment["namespace"].(string); ok {
		properties["namespace"] = namespace
	}
	if labels, ok := deployment["labels"].(map[string]interface{}); ok {
		properties["has_labels"] = len(labels) > 0
		properties["label_count"] = len(labels)
	}
	if annotations, ok := deployment["annotations"].(map[string]interface{}); ok {
		properties["has_annotations"] = len(annotations) > 0
		properties["annotation_count"] = len(annotations)
	}

	// Extract spec information
	if spec, ok := deployment["spec"].(map[string]any); ok {
		if replicas, ok := spec["replicas"]; ok {
			properties["spec_replicas"] = replicas
		}
		if selector, ok := spec["selector"].(map[string]string); ok {
			properties["has_selector"] = len(selector) > 0
			properties["selector_count"] = len(selector)

			var selectors []string
			for key, value := range selector {
				selectors = append(selectors, fmt.Sprintf("%s=%s", key, value))
			}
			if len(selectors) > 0 {
				properties["spec_selector_match_labels"] = strings.Join(selectors, ",")
			}
		}
		if strategyType, ok := spec["strategy_type"].(string); ok {
			properties["spec_strategy_type"] = strategyType
			properties["is_rolling_update"] = strategyType == "RollingUpdate"
			properties["is_recreate"] = strategyType == "Recreate"
		}
		if revisionHistoryLimit, ok := spec["revision_history_limit"]; ok {
			properties["spec_revision_history_limit"] = revisionHistoryLimit
		}
		if progressDeadlineSeconds, ok := spec["progress_deadline_seconds"]; ok {
			properties["spec_progress_deadline_seconds"] = progressDeadlineSeconds
		}
		if containerImages, ok := spec["container_images"].([]string); ok {
			properties["spec_template_images"] = strings.Join(containerImages, ",")
		}
	}

	// Extract status information
	if status, ok := deployment["status"].(map[string]any); ok {
		if replicas, ok := status["replicas"]; ok {
			properties["status_replicas"] = replicas
		}
		if readyReplicas, ok := status["ready_replicas"]; ok {
			properties["status_ready_replicas"] = readyReplicas
		}
		if availableReplicas, ok := status["available_replicas"]; ok {
			properties["status_available_replicas"] = availableReplicas
		}
		if unavailableReplicas, ok := status["unavailable_replicas"]; ok {
			properties["status_unavailable_replicas"] = unavailableReplicas
		}
		if updatedReplicas, ok := status["updated_replicas"]; ok {
			properties["status_updated_replicas"] = updatedReplicas
		}
		if observedGeneration, ok := status["observed_generation"]; ok {
			properties["status_observed_generation"] = observedGeneration
		}
	}

	return properties, nil
}

type DeploymentParser struct {
	config bloodhound.ResourceConfig
}

func NewDeploymentParser() *DeploymentParser {
	return &DeploymentParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "deployment",
			PrimaryKind:    "Deployment",
			SecondaryKinds: []string{},
			PropertyMapper: &DeploymentPropertyMapper{},
		},
	}
}

func (p *DeploymentParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *DeploymentParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *DeploymentParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *DeploymentParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	deploymentData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment resource: %w", err)
	}

	var deployment map[string]any
	if err := json.Unmarshal(deploymentData, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	var name string
	if metadata, ok := deployment["metadata"].(map[string]any); ok && metadata != nil {
		name, _ = metadata["name"].(string)
	}

	bhNode, err := bloodhound.CreateNodeWithConfig(
		p.config,
		resource.Type,
		resource.Namespace,
		name,
		resource.Resource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment node: %w", err)
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}
