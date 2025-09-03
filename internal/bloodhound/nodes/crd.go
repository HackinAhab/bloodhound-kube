package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"bloodhound-kube/internal/bloodhound"
	"bloodhound-kube/internal/collector"
)

type CRDPropertyMapper struct{}

func (m *CRDPropertyMapper) MapProperties(resource any) (map[string]any, error) {
	crdData, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CRD resource: %w", err)
	}

	var crd collector.CRD
	if err := json.Unmarshal(crdData, &crd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CRD: %w", err)
	}

	properties := map[string]any{
		"name":       crd.Name,
		"group":      crd.Group,
		"kind":       crd.Kind,
		"version":    crd.Version,
		"scope":      crd.Scope,
		"plural":     crd.Plural,
		"singular":   crd.Singular,
		"created_at": crd.CreatedAt,

		"api_group_path": fmt.Sprintf("apis/%s/%s", crd.Group, crd.Version),
		"resource_path":  fmt.Sprintf("/apis/%s/%s/%s", crd.Group, crd.Version, crd.Plural),

		"is_cluster_scoped":   crd.Scope == "Cluster",
		"is_namespace_scoped": crd.Scope == "Namespaced",

		"versions_count":     len(crd.Versions),
		"has_multiple_versions": len(crd.Versions) > 1,
	}

	if len(crd.ShortNames) > 0 {
		properties["short_names"] = crd.ShortNames
		properties["has_short_names"] = true
		properties["short_names_count"] = len(crd.ShortNames)
	} else {
		properties["has_short_names"] = false
		properties["short_names_count"] = 0
	}

	if len(crd.Categories) > 0 {
		properties["categories"] = crd.Categories
		properties["has_categories"] = true
		properties["categories_count"] = len(crd.Categories)
		
		var highPrivCategories []string
		for _, category := range crd.Categories {
			lowerCat := strings.ToLower(category)
			if strings.Contains(lowerCat, "admin") ||
			   strings.Contains(lowerCat, "security") ||
			   strings.Contains(lowerCat, "policy") ||
			   strings.Contains(lowerCat, "rbac") ||
			   strings.Contains(lowerCat, "auth") {
				highPrivCategories = append(highPrivCategories, category)
			}
		}
		
		if len(highPrivCategories) > 0 {
			properties["high_privilege_categories"] = highPrivCategories
			properties["has_high_privilege_categories"] = true
		}
	} else {
		properties["has_categories"] = false
		properties["categories_count"] = 0
	}

	var activeVersions []string
	var storageVersions []string
	var deprecatedVersions []string
	
	for _, version := range crd.Versions {
		if version.Served {
			activeVersions = append(activeVersions, version.Name)
		} else {
			deprecatedVersions = append(deprecatedVersions, version.Name)
		}
		
		if version.Storage {
			storageVersions = append(storageVersions, version.Name)
		}
	}
	
	properties["active_versions"] = activeVersions
	properties["active_versions_count"] = len(activeVersions)
	properties["storage_versions"] = storageVersions
	properties["storage_versions_count"] = len(storageVersions)
	
	if len(deprecatedVersions) > 0 {
		properties["deprecated_versions"] = deprecatedVersions
		properties["has_deprecated_versions"] = true
		properties["deprecated_versions_count"] = len(deprecatedVersions)
	}

	if crd.Labels != nil {
		labelCount := len(crd.Labels)
		properties["labels_count"] = labelCount
		
		var securityLabels []string
		var operatorLabels []string
		var vendorLabels []string
		
		for key, value := range crd.Labels {
			lowerKey := strings.ToLower(key)
			lowerValue := strings.ToLower(value)
			
			if strings.Contains(lowerKey, "security") ||
			   strings.Contains(lowerKey, "policy") ||
			   strings.Contains(lowerKey, "rbac") ||
			   strings.Contains(lowerKey, "auth") {
				securityLabels = append(securityLabels, key+"="+value)
			}
			
			if strings.Contains(lowerKey, "operator") ||
			   strings.Contains(lowerValue, "operator") {
				operatorLabels = append(operatorLabels, key+"="+value)
			}
			
			if strings.Contains(lowerKey, "vendor") ||
			   strings.Contains(lowerKey, "app.kubernetes.io/managed-by") ||
			   strings.Contains(lowerKey, "helm.sh") {
				vendorLabels = append(vendorLabels, key+"="+value)
			}
		}
		
		if len(securityLabels) > 0 {
			properties["security_labels"] = securityLabels
			properties["has_security_labels"] = true
		}
		
		if len(operatorLabels) > 0 {
			properties["operator_labels"] = operatorLabels
			properties["is_operator_managed"] = true
		}
		
		if len(vendorLabels) > 0 {
			properties["vendor_labels"] = vendorLabels
			properties["has_vendor_info"] = true
		}
	}

	if crd.Annotations != nil {
		annotationCount := len(crd.Annotations)
		properties["annotations_count"] = annotationCount
		
		securityAnnotations := make(map[string]any)
		var certManagerAnnotations []string
		var helmAnnotations []string
		
		for key, value := range crd.Annotations {
			lowerKey := strings.ToLower(key)
			
			if strings.Contains(lowerKey, "security") ||
			   strings.Contains(lowerKey, "policy") ||
			   strings.Contains(lowerKey, "rbac") {
				securityAnnotations[key] = value
			}
			
			if strings.Contains(lowerKey, "cert-manager") ||
			   strings.Contains(lowerKey, "certificate") {
				certManagerAnnotations = append(certManagerAnnotations, key+"="+value)
			}
			
			if strings.Contains(lowerKey, "helm.sh") ||
			   strings.Contains(lowerKey, "meta.helm.sh") {
				helmAnnotations = append(helmAnnotations, key+"="+value)
			}
		}
		
		if len(securityAnnotations) > 0 {
			properties["security_annotations"] = securityAnnotations
			properties["has_security_annotations"] = true
		}
		
		if len(certManagerAnnotations) > 0 {
			properties["cert_manager_annotations"] = certManagerAnnotations
			properties["manages_certificates"] = true
		}
		
		if len(helmAnnotations) > 0 {
			properties["helm_annotations"] = helmAnnotations
			properties["is_helm_managed"] = true
		}
	}

	if len(crd.Conditions) > 0 {
		var conditionDetails []map[string]any
		var problemConditions []string
		var establishedCondition bool
		var namesAcceptedCondition bool
		
		for _, condition := range crd.Conditions {
			conditionInfo := map[string]any{
				"type":    condition.Type,
				"status":  condition.Status,
				"reason":  condition.Reason,
				"message": condition.Message,
			}
			conditionDetails = append(conditionDetails, conditionInfo)
			
			switch condition.Type {
			case "Established":
				isEstablished := condition.Status == "True"
				establishedCondition = isEstablished
				if !isEstablished {
					problemConditions = append(problemConditions, "not_established")
				}
			case "NamesAccepted":
				namesAccepted := condition.Status == "True"
				namesAcceptedCondition = namesAccepted
				if !namesAccepted {
					problemConditions = append(problemConditions, "names_not_accepted")
				}
			}
		}
		
		properties["condition_details"] = conditionDetails
		properties["conditions_count"] = len(crd.Conditions)
		properties["is_established"] = establishedCondition
		properties["names_accepted"] = namesAcceptedCondition
		
		if len(problemConditions) > 0 {
			properties["problem_conditions"] = problemConditions
			properties["has_problems"] = true
		} else {
			properties["has_problems"] = false
		}
	}

	securityScore := 0
	var securityIssues []string
	var attackSurface []string

	if properties["is_cluster_scoped"] == true {
		securityScore += 8
		securityIssues = append(securityIssues, "cluster_scoped_permissions")
		attackSurface = append(attackSurface, "cluster_admin_access")
	}

	if properties["has_security_labels"] == true || properties["has_security_annotations"] == true {
		securityScore += 6
		securityIssues = append(securityIssues, "security_related_resource")
		attackSurface = append(attackSurface, "security_policy_control")
	}

	if properties["is_operator_managed"] == true {
		securityScore += 5
		securityIssues = append(securityIssues, "operator_managed")
		attackSurface = append(attackSurface, "operator_privileges")
	}

	if properties["has_multiple_versions"] == true {
		securityScore += 3
		securityIssues = append(securityIssues, "multiple_api_versions")
		attackSurface = append(attackSurface, "version_confusion")
	}

	if properties["manages_certificates"] == true {
		securityScore += 7
		securityIssues = append(securityIssues, "certificate_management")
		attackSurface = append(attackSurface, "pki_infrastructure")
	}

	if properties["has_deprecated_versions"] == true {
		securityScore += 4
		securityIssues = append(securityIssues, "deprecated_versions")
		attackSurface = append(attackSurface, "legacy_api_access")
	}

	if properties["has_short_names"] == true {
		securityScore += 2
		attackSurface = append(attackSurface, "alternate_api_names")
	}

	properties["security_score"] = securityScore
	properties["is_high_value_target"] = securityScore >= 10
	properties["is_security_relevant"] = securityScore >= 5

	if len(securityIssues) > 0 {
		properties["security_issues"] = securityIssues
	}

	if len(attackSurface) > 0 {
		properties["attack_surface"] = attackSurface
	}

	// API access patterns for reconnaissance
	properties["kubectl_access_names"] = []string{crd.Plural, crd.Kind}
	if len(crd.ShortNames) > 0 {
		accessNames := append([]string{crd.Plural, crd.Kind}, crd.ShortNames...)
		properties["kubectl_access_names"] = accessNames
	}

	var kubectlCommands []string
	baseCmd := fmt.Sprintf("kubectl get %s", crd.Plural)
	kubectlCommands = append(kubectlCommands, baseCmd)
	
	if crd.Scope == "Namespaced" {
		kubectlCommands = append(kubectlCommands, baseCmd+" -A")
		kubectlCommands = append(kubectlCommands, baseCmd+" -n <namespace>")
	}
	
	if len(crd.ShortNames) > 0 {
		kubectlCommands = append(kubectlCommands, fmt.Sprintf("kubectl get %s", crd.ShortNames[0]))
	}
	
	properties["kubectl_commands"] = kubectlCommands

	return properties, nil
}

type CRDParser struct {
	config bloodhound.ResourceConfig
}

func NewCRDParser() *CRDParser {
	return &CRDParser{
		config: bloodhound.ResourceConfig{
			ResourceType:   "crd",
			PrimaryKind:    "CustomResourceDefinition",
			SecondaryKinds: []string{"CRD", "APIExtension"},
			PropertyMapper: &CRDPropertyMapper{},
		},
	}
}

func (p *CRDParser) GetResourceType() string {
	return p.config.ResourceType
}

func (p *CRDParser) GetSupportedKinds() []string {
	kinds := []string{p.config.PrimaryKind}
	return append(kinds, p.config.SecondaryKinds...)
}

func (p *CRDParser) GetConfig() bloodhound.ResourceConfig {
	return p.config
}

func (p *CRDParser) Parse(resource bloodhound.ResourceData) (*bloodhound.ParsedResult, error) {
	crdData, err := json.Marshal(resource.Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CRD resource: %w", err)
	}

	var crd collector.CRD
	if err := json.Unmarshal(crdData, &crd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CRD: %w", err)
	}

	bhNode, err := bloodhound.CreateNodeWithConfig(
		p.config,
		resource.Type,
		resource.Namespace,
		crd.Name,
		resource.Resource,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CRD node: %w", err)
	}

	result := &bloodhound.ParsedResult{
		Nodes: []bloodhound.BloodHoundNode{bhNode},
		Edges: []bloodhound.BloodHoundEdge{},
	}

	return result, nil
}