package report

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"bloodhound-kube/internal/utils"
)

// NewGenerator creates a new report generator
func NewGenerator(config Config, log *utils.Logger) (*Generator, error) {
	g := &Generator{
		config: config,
		log:    log,
		data: &CollectedData{
			Namespaces: make(map[string]*Namespace),
		},
	}

	if err := g.loadData(); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	return g, nil
}

// Generate creates reports based on the configuration
func (g *Generator) Generate() ([]*Report, error) {
	var allReports []*Report

	for _, reportType := range g.config.ReportTypes {
		var reports []*Report
		var err error

		switch reportType {
		case "privileged":
			reports, err = g.generatePrivilegedReport()
		case "privesc":
			reports, err = g.generatePrivilegeEscalationReport()
		case "nonroot":
			reports, err = g.generateNonRootReport()
		case "caps":
			reports, err = g.generateCapabilitiesReport()
		case "imgsrc":
			reports, err = g.generateImageSourceReport()
		case "seccomp":
			reports, err = g.generateSeccompReport()
		case "limits":
			reports, err = g.generateLimitsReport()
		case "serviceaccount":
			reports, err = g.generateServiceAccountReport()
		case "token":
			reports, err = g.generateTokenReport()
		case "all":
			reports, err = g.generateAllReports()
		default:
			return nil, fmt.Errorf("unknown report type: %s", reportType)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to generate %s report: %w", reportType, err)
		}

		allReports = append(allReports, reports...)
	}

	return allReports, nil
}

// loadData reads the JSONL file and populates the data structures
func (g *Generator) loadData() error {
	file, err := os.Open(g.config.InputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var rawItem map[string]any
		if err := json.Unmarshal([]byte(line), &rawItem); err != nil {
			g.log.Debug("Failed to parse JSON line", "line", lineNum, "error", err)
			continue
		}

		itemType, ok := rawItem["type"].(string)
		if !ok {
			g.log.Debug("Missing or invalid type field", "line", lineNum)
			continue
		}

		if err := g.processItem(itemType, rawItem); err != nil {
			g.log.Debug("Failed to process item", "type", itemType, "line", lineNum, "error", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	g.log.Info("Loaded data", "namespaces", len(g.data.Namespaces))
	return nil
}

// processItem processes a single item from the JSONL file
func (g *Generator) processItem(itemType string, rawItem map[string]interface{}) error {
	switch itemType {
	case "pod":
		return g.processPod(rawItem)
	case "secret":
		return g.processSecret(rawItem)
	case "rbac":
		return g.processRBAC(rawItem)
	default:
		// Skip unknown types
		return nil
	}
}

// getOrCreateNamespace gets or creates a namespace
func (g *Generator) getOrCreateNamespace(name string) *Namespace {
	if ns, exists := g.data.Namespaces[name]; exists {
		return ns
	}

	ns := &Namespace{
		Name:            name,
		Pods:            make([]*Pod, 0),
		Secrets:         make([]*Secret, 0),
		ServiceAccounts: make([]*ServiceAccount, 0),
		RBAC:            make([]*RBACResource, 0),
	}
	g.data.Namespaces[name] = ns
	return ns
}

// processPod processes a pod from the JSONL
func (g *Generator) processPod(rawItem map[string]any) error {
	resource, ok := rawItem["resource"].(map[string]any)
	if !ok {
		return fmt.Errorf("invalid pod resource")
	}

	name, _ := resource["name"].(string)
	namespace, _ := resource["namespace"].(string)
	nodeName, _ := resource["node_name"].(string)
	hostNetwork, _ := resource["host_network"].(bool)
	serviceAccount, _ := resource["service_account"].(string)

	var createdAt time.Time
	if createdAtStr, ok := resource["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			createdAt = t
		}
	}

	labels := make(map[string]string)
	if labelMap, ok := resource["labels"].(map[string]any); ok {
		for k, v := range labelMap {
			if str, ok := v.(string); ok {
				labels[k] = str
			}
		}
	}

	containers := make([]*Container, 0)
	if containerList, ok := resource["containers"].([]any); ok {
		for _, c := range containerList {
			if containerMap, ok := c.(map[string]any); ok {
				container := g.parseContainer(containerMap)
				if container != nil {
					containers = append(containers, container)
				}
			}
		}
	}

	pod := &Pod{
		Name:           name,
		Namespace:      namespace,
		NodeName:       nodeName,
		HostNetwork:    hostNetwork,
		ServiceAccount: serviceAccount,
		Containers:     containers,
		CreatedAt:      createdAt,
		Labels:         labels,
	}

	if cpuReq, ok := resource["cpu_request"].(string); ok {
		pod.CPURequest = cpuReq
	}
	if cpuLimit, ok := resource["cpu_limit"].(string); ok {
		pod.CPULimit = cpuLimit
	}
	if memReq, ok := resource["memory_request"].(string); ok {
		pod.MemoryRequest = memReq
	}
	if memLimit, ok := resource["memory_limit"].(string); ok {
		pod.MemoryLimit = memLimit
	}

	// Parse pod-level security context
	if secCtx, ok := resource["security_context"].(map[string]any); ok {
		pod.SecurityContext = g.parseSecurityContext(secCtx)
	}

	ns := g.getOrCreateNamespace(namespace)
	ns.Pods = append(ns.Pods, pod)

	return nil
}

// parseContainer parses a container from the resource
func (g *Generator) parseContainer(containerMap map[string]any) *Container {
	name, _ := containerMap["name"].(string)
	image, _ := containerMap["image"].(string)

	container := &Container{
		Name:  name,
		Image: image,
	}

	// Parse resource limits/requests
	if cpuReq, ok := containerMap["cpu_request"].(string); ok {
		container.CPURequest = cpuReq
	}
	if cpuLimit, ok := containerMap["cpu_limit"].(string); ok {
		container.CPULimit = cpuLimit
	}
	if memReq, ok := containerMap["memory_request"].(string); ok {
		container.MemoryRequest = memReq
	}
	if memLimit, ok := containerMap["memory_limit"].(string); ok {
		container.MemoryLimit = memLimit
	}

	// Parse security context
	if secCtx, ok := containerMap["security_context"].(map[string]any); ok {
		container.SecurityContext = g.parseSecurityContext(secCtx)
	}

	return container
}

// parseSecurityContext parses security context from map
func (g *Generator) parseSecurityContext(secCtx map[string]any) *SecurityContext {
	sc := &SecurityContext{}

	if runAsUser, ok := secCtx["run_as_user"].(float64); ok {
		uid := int64(runAsUser)
		sc.RunAsUser = &uid
	}

	if runAsNonRoot, ok := secCtx["run_as_non_root"].(bool); ok {
		sc.RunAsNonRoot = &runAsNonRoot
	}

	if allowPrivEsc, ok := secCtx["allow_priv_esc"].(bool); ok {
		sc.AllowPrivilegeEscalation = &allowPrivEsc
	}

	if privileged, ok := secCtx["privileged"].(bool); ok {
		sc.Privileged = &privileged
	}

	if capsMap, ok := secCtx["linux_capabilities"].(map[string]interface{}); ok {
		caps := &Capabilities{}
		if addList, ok := capsMap["capabilities_add"].([]interface{}); ok {
			for _, cap := range addList {
				if capStr, ok := cap.(string); ok {
					caps.Add = append(caps.Add, capStr)
				}
			}
		}
		if dropList, ok := capsMap["capabilities_drop"].([]interface{}); ok {
			for _, cap := range dropList {
				if capStr, ok := cap.(string); ok {
					caps.Drop = append(caps.Drop, capStr)
				}
			}
		}
		sc.Capabilities = caps
	}

	if seccomp, ok := secCtx["seccomp_profile"].(string); ok {
		sc.SeccompProfile = seccomp
	}

	return sc
}

// processSecret processes a secret from the JSONL
func (g *Generator) processSecret(rawItem map[string]any) error {
	resource, ok := rawItem["resource"].(map[string]any)
	if !ok {
		return fmt.Errorf("invalid secret resource")
	}

	name, _ := resource["name"].(string)
	namespace, _ := resource["namespace"].(string)
	secretType, _ := resource["type"].(string)

	var createdAt time.Time
	if createdAtStr, ok := resource["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			createdAt = t
		}
	}

	// Parse data keys
	var dataKeys []string
	if keys, ok := resource["data_keys"].([]any); ok {
		for _, key := range keys {
			if keyStr, ok := key.(string); ok {
				dataKeys = append(dataKeys, keyStr)
			}
		}
	}

	// Parse data (for token extraction)
	data := make(map[string]string)
	if dataMap, ok := resource["data"].(map[string]any); ok {
		for k, v := range dataMap {
			if str, ok := v.(string); ok {
				data[k] = str
			}
		}
	}

	secret := &Secret{
		Name:      name,
		Namespace: namespace,
		Type:      secretType,
		DataKeys:  dataKeys,
		Data:      data,
		CreatedAt: createdAt,
	}

	ns := g.getOrCreateNamespace(namespace)
	ns.Secrets = append(ns.Secrets, secret)

	// If this is a service account token, also create/update the service account
	if strings.Contains(secretType, "service-account-token") {
		g.processServiceAccountToken(secret)
	}

	return nil
}

// processServiceAccountToken processes a service account token from a secret
func (g *Generator) processServiceAccountToken(secret *Secret) {
	// Extract service account name from annotations or name
	var saName string
	// This would need to be extracted from metadata.annotations["kubernetes.io/service-account.name"]
	// For now, we'll derive it from the secret name pattern
	if strings.HasSuffix(secret.Name, "-token") {
		saName = strings.TrimSuffix(secret.Name, "-token")
	} else {
		saName = secret.Name
	}

	ns := g.getOrCreateNamespace(secret.Namespace)

	// Find existing service account or create new one
	var sa *ServiceAccount
	for _, existing := range ns.ServiceAccounts {
		if existing.Name == saName {
			sa = existing
			break
		}
	}

	if sa == nil {
		sa = &ServiceAccount{
			Name:      saName,
			Namespace: secret.Namespace,
			CreatedAt: secret.CreatedAt,
		}
		ns.ServiceAccounts = append(ns.ServiceAccounts, sa)
	}

	// Add token data if available
	if token, ok := secret.Data["token"]; ok {
		sa.Token = token
	}
}

// processRBAC processes RBAC resources from the JSONL
func (g *Generator) processRBAC(rawItem map[string]any) error {
	resource, ok := rawItem["resource"].(map[string]any)
	if !ok {
		return fmt.Errorf("invalid rbac resource")
	}

	name, _ := resource["name"].(string)
	namespace, _ := resource["namespace"].(string) // May be empty for cluster resources
	kind, _ := resource["kind"].(string)

	var createdAt time.Time
	if createdAtStr, ok := resource["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			createdAt = t
		}
	}

	rbac := &RBACResource{
		Name:      name,
		Namespace: namespace,
		Kind:      kind,
		CreatedAt: createdAt,
	}

	// Parse rules if present (for Roles/ClusterRoles)
	if rulesList, ok := resource["rules"].([]interface{}); ok {
		for _, r := range rulesList {
			if ruleMap, ok := r.(map[string]interface{}); ok {
				rule := PolicyRule{}

				if apiGroups, ok := ruleMap["api_groups"].([]interface{}); ok {
					for _, ag := range apiGroups {
						if agStr, ok := ag.(string); ok {
							rule.APIGroups = append(rule.APIGroups, agStr)
						}
					}
				}

				if resources, ok := ruleMap["resources"].([]interface{}); ok {
					for _, res := range resources {
						if resStr, ok := res.(string); ok {
							rule.Resources = append(rule.Resources, resStr)
						}
					}
				}

				if verbs, ok := ruleMap["verbs"].([]interface{}); ok {
					for _, verb := range verbs {
						if verbStr, ok := verb.(string); ok {
							rule.Verbs = append(rule.Verbs, verbStr)
						}
					}
				}

				if resourceNames, ok := ruleMap["resource_names"].([]interface{}); ok {
					for _, rn := range resourceNames {
						if rnStr, ok := rn.(string); ok {
							rule.ResourceNames = append(rule.ResourceNames, rnStr)
						}
					}
				}

				rbac.Rules = append(rbac.Rules, rule)
			}
		}
	}

	// Parse subjects if present (for bindings)
	if subjectsList, ok := resource["subjects"].([]any); ok {
		for _, s := range subjectsList {
			if subjectMap, ok := s.(map[string]any); ok {
				subject := Subject{}
				subject.Kind, _ = subjectMap["kind"].(string)
				subject.Name, _ = subjectMap["name"].(string)
				subject.Namespace, _ = subjectMap["namespace"].(string)
				rbac.Subjects = append(rbac.Subjects, subject)
			}
		}
	}

	// Parse role ref if present (for bindings)
	if roleRefMap, ok := resource["role_ref"].(map[string]any); ok {
		roleRef := &RoleRef{}
		roleRef.Kind, _ = roleRefMap["kind"].(string)
		roleRef.Name, _ = roleRefMap["name"].(string)
		roleRef.APIGroup, _ = roleRefMap["api_group"].(string)
		rbac.RoleRef = roleRef
	}

	// Add to appropriate namespace or global if cluster resource
	if namespace != "" {
		ns := g.getOrCreateNamespace(namespace)
		ns.RBAC = append(ns.RBAC, rbac)
	} else {
		// For cluster resources, we'll add to a special "cluster" namespace
		ns := g.getOrCreateNamespace("cluster")
		ns.RBAC = append(ns.RBAC, rbac)
	}

	return nil
}
