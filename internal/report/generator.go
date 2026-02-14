package report

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// NewGenerator creates a new report generator
func NewGenerator(config Config, log logrus.FieldLogger) (*Generator, error) {
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

	reader := bufio.NewReader(file)
	lineNum := 0

	for {
		lineBytes, err := reader.ReadBytes('\n')
		if err != nil && len(lineBytes) == 0 {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading file: %w", err)
		}

		lineNum++
		line := strings.TrimSpace(string(lineBytes))
		if line == "" {
			if err == io.EOF {
				break
			}
			continue
		}

		var rawItem map[string]any
		if err := json.Unmarshal([]byte(line), &rawItem); err != nil {
			g.log.WithError(err).WithField("line", lineNum).Debug("Failed to parse JSON line")
			if err == io.EOF {
				break
			}
			continue
		}

		itemType, ok := rawItem["type"].(string)
		if !ok {
			g.log.WithField("line", lineNum).Debug("Missing or invalid type field")
			if err == io.EOF {
				break
			}
			continue
		}

		if err := g.processItem(itemType, rawItem); err != nil {
			g.log.WithError(err).WithFields(logrus.Fields{"type": itemType, "line": lineNum}).Debug("Failed to process item")
		}

		if err == io.EOF {
			break
		}
	}

	g.log.WithField("namespaces", len(g.data.Namespaces)).Info("Loaded data")
	return nil
}

// processItem processes a single item from the JSONL file
func (g *Generator) processItem(itemType string, rawItem map[string]interface{}) error {
	resource, _ := rawItem["resource"].(map[string]any)
	kind := strings.ToLower(getString(resource, "kind"))
	typeKey := strings.ToLower(itemType)

	switch {
	case kind == "pod" || typeKey == "pod" || typeKey == "pods":
		return g.processPod(resource)
	case kind == "secret" || typeKey == "secret" || typeKey == "secrets":
		return g.processSecret(resource)
	case kind == "role" || kind == "clusterrole" || kind == "rolebinding" || kind == "clusterrolebinding" ||
		typeKey == "role" || typeKey == "roles" || typeKey == "clusterrole" || typeKey == "clusterroles" ||
		typeKey == "rolebinding" || typeKey == "rolebindings" || typeKey == "clusterrolebinding" || typeKey == "clusterrolebindings":
		return g.processRBAC(resource)
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
func (g *Generator) processPod(resource map[string]any) error {
	if resource == nil {
		return fmt.Errorf("invalid pod resource")
	}

	metadata := getMap(resource, "metadata")
	spec := getMap(resource, "spec")

	name := getString(metadata, "name")
	namespace := getString(metadata, "namespace")
	nodeName := getString(spec, "nodeName")
	hostNetwork := getBool(spec, "hostNetwork")
	serviceAccount := getString(spec, "serviceAccountName")

	var createdAt time.Time
	if createdAtStr := getString(metadata, "creationTimestamp"); createdAtStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			createdAt = t
		}
	}

	labels := getStringMap(metadata, "labels")

	containers := make([]*Container, 0)
	if containerList, ok := spec["containers"].([]any); ok {
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

	if secCtx, ok := spec["securityContext"].(map[string]any); ok {
		pod.SecurityContext = g.parseSecurityContext(secCtx)
	}

	ns := g.getOrCreateNamespace(namespace)
	ns.Pods = append(ns.Pods, pod)

	return nil
}

// parseContainer parses a container from the resource
func (g *Generator) parseContainer(containerMap map[string]any) *Container {
	name := getString(containerMap, "name")
	image := getString(containerMap, "image")

	container := &Container{
		Name:  name,
		Image: image,
	}

	// Parse resource limits/requests
	if resources, ok := containerMap["resources"].(map[string]any); ok {
		if requests, ok := resources["requests"].(map[string]any); ok {
			if cpuReq := getString(requests, "cpu"); cpuReq != "" {
				container.CPURequest = cpuReq
			}
			if memReq := getString(requests, "memory"); memReq != "" {
				container.MemoryRequest = memReq
			}
		}
		if limits, ok := resources["limits"].(map[string]any); ok {
			if cpuLimit := getString(limits, "cpu"); cpuLimit != "" {
				container.CPULimit = cpuLimit
			}
			if memLimit := getString(limits, "memory"); memLimit != "" {
				container.MemoryLimit = memLimit
			}
		}
	}

	// Parse security context
	if secCtx, ok := containerMap["securityContext"].(map[string]any); ok {
		container.SecurityContext = g.parseSecurityContext(secCtx)
	}

	return container
}

// parseSecurityContext parses security context from map
func (g *Generator) parseSecurityContext(secCtx map[string]any) *SecurityContext {
	sc := &SecurityContext{}

	if runAsUser := getInt64(secCtx, "runAsUser"); runAsUser != nil {
		sc.RunAsUser = runAsUser
	}

	if runAsNonRoot, ok := secCtx["runAsNonRoot"].(bool); ok {
		sc.RunAsNonRoot = &runAsNonRoot
	}

	if allowPrivEsc, ok := secCtx["allowPrivilegeEscalation"].(bool); ok {
		sc.AllowPrivilegeEscalation = &allowPrivEsc
	}

	if privileged, ok := secCtx["privileged"].(bool); ok {
		sc.Privileged = &privileged
	}

	if capsMap, ok := secCtx["capabilities"].(map[string]interface{}); ok {
		caps := &Capabilities{}
		if addList, ok := capsMap["add"].([]interface{}); ok {
			for _, cap := range addList {
				if capStr, ok := cap.(string); ok {
					caps.Add = append(caps.Add, capStr)
				}
			}
		}
		if dropList, ok := capsMap["drop"].([]interface{}); ok {
			for _, cap := range dropList {
				if capStr, ok := cap.(string); ok {
					caps.Drop = append(caps.Drop, capStr)
				}
			}
		}
		sc.Capabilities = caps
	}

	if seccompMap, ok := secCtx["seccompProfile"].(map[string]any); ok {
		seccompType := getString(seccompMap, "type")
		if seccompType == "Localhost" {
			if localhost := getString(seccompMap, "localhostProfile"); localhost != "" {
				sc.SeccompProfile = fmt.Sprintf("Localhost:%s", localhost)
			} else {
				sc.SeccompProfile = seccompType
			}
		} else {
			sc.SeccompProfile = seccompType
		}
	}

	return sc
}

// processSecret processes a secret from the JSONL
func (g *Generator) processSecret(resource map[string]any) error {
	if resource == nil {
		return fmt.Errorf("invalid secret resource")
	}

	metadata := getMap(resource, "metadata")

	name := getString(metadata, "name")
	namespace := getString(metadata, "namespace")
	secretType := getString(resource, "type")

	var createdAt time.Time
	if createdAtStr := getString(metadata, "creationTimestamp"); createdAtStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			createdAt = t
		}
	}

	// Parse data keys
	var dataKeys []string
	data := make(map[string]string)
	if dataMap, ok := resource["data"].(map[string]any); ok {
		for k, v := range dataMap {
			dataKeys = append(dataKeys, k)
			if k == "token" {
				if str, ok := v.(string); ok {
					decoded, err := base64.StdEncoding.DecodeString(str)
					if err != nil {
						g.log.WithError(err).WithFields(logrus.Fields{"secret": name, "namespace": namespace}).Debug("Failed to decode secret token")
						data[k] = str
					} else {
						data[k] = string(decoded)
					}
				}
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
		annotations := getStringMap(metadata, "annotations")
		g.processServiceAccountToken(secret, annotations)
	}

	return nil
}

// processServiceAccountToken processes a service account token from a secret
func (g *Generator) processServiceAccountToken(secret *Secret, annotations map[string]string) {
	var saName string
	if annotations != nil {
		if name, ok := annotations["kubernetes.io/service-account.name"]; ok {
			saName = name
		}
	}
	if saName == "" {
		if strings.HasSuffix(secret.Name, "-token") {
			saName = strings.TrimSuffix(secret.Name, "-token")
		} else {
			saName = secret.Name
		}
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
func (g *Generator) processRBAC(resource map[string]any) error {
	if resource == nil {
		return fmt.Errorf("invalid rbac resource")
	}

	metadata := getMap(resource, "metadata")

	name := getString(metadata, "name")
	namespace := getString(metadata, "namespace") // May be empty for cluster resources
	kind := getString(resource, "kind")

	var createdAt time.Time
	if createdAtStr := getString(metadata, "creationTimestamp"); createdAtStr != "" {
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
	if rulesList, ok := resource["rules"].([]any); ok {
		for _, r := range rulesList {
			if ruleMap, ok := r.(map[string]any); ok {
				rule := PolicyRule{}

				if apiGroups, ok := ruleMap["apiGroups"].([]any); ok {
					for _, ag := range apiGroups {
						if agStr, ok := ag.(string); ok {
							rule.APIGroups = append(rule.APIGroups, agStr)
						}
					}
				}

				if resources, ok := ruleMap["resources"].([]any); ok {
					for _, res := range resources {
						if resStr, ok := res.(string); ok {
							rule.Resources = append(rule.Resources, resStr)
						}
					}
				}

				if verbs, ok := ruleMap["verbs"].([]any); ok {
					for _, verb := range verbs {
						if verbStr, ok := verb.(string); ok {
							rule.Verbs = append(rule.Verbs, verbStr)
						}
					}
				}

				if resourceNames, ok := ruleMap["resourceNames"].([]any); ok {
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
				subject.Kind = getString(subjectMap, "kind")
				subject.Name = getString(subjectMap, "name")
				subject.Namespace = getString(subjectMap, "namespace")
				rbac.Subjects = append(rbac.Subjects, subject)
			}
		}
	}

	// Parse role ref if present (for bindings)
	if roleRefMap, ok := resource["roleRef"].(map[string]any); ok {
		roleRef := &RoleRef{}
		roleRef.Kind = getString(roleRefMap, "kind")
		roleRef.Name = getString(roleRefMap, "name")
		roleRef.APIGroup = getString(roleRefMap, "apiGroup")
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

func getMap(obj map[string]any, key string) map[string]any {
	if obj == nil {
		return nil
	}
	if value, ok := obj[key].(map[string]any); ok {
		return value
	}
	return nil
}

func getString(obj map[string]any, key string) string {
	if obj == nil {
		return ""
	}
	if value, ok := obj[key].(string); ok {
		return value
	}
	return ""
}

func getBool(obj map[string]any, key string) bool {
	if obj == nil {
		return false
	}
	if value, ok := obj[key].(bool); ok {
		return value
	}
	return false
}

func getInt64(obj map[string]any, key string) *int64 {
	if obj == nil {
		return nil
	}
	switch value := obj[key].(type) {
	case float64:
		v := int64(value)
		return &v
	case int64:
		v := value
		return &v
	case int:
		v := int64(value)
		return &v
	case json.Number:
		if v, err := value.Int64(); err == nil {
			return &v
		}
	}
	return nil
}

func getStringMap(obj map[string]any, key string) map[string]string {
	if obj == nil {
		return nil
	}
	raw, ok := obj[key].(map[string]any)
	if !ok {
		return nil
	}
	mapped := make(map[string]string, len(raw))
	for k, v := range raw {
		if str, ok := v.(string); ok {
			mapped[k] = str
		}
	}
	if len(mapped) == 0 {
		return nil
	}
	return mapped
}
