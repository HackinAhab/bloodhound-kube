package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// generatePrivilegedReport generates a report of privileged containers
func (g *Generator) generatePrivilegedReport() ([]*Report, error) {
	reportData := make([]*ReportNamespace, 0)

	for _, namespace := range g.data.Namespaces {
		nsReport := &ReportNamespace{
			Namespace: namespace.Name,
			Pods:      make([]any, 0),
		}

		for _, pod := range namespace.Pods {
			podReport := &PodReport{
				Name:       pod.Name,
				Containers: make([]any, 0),
			}

			hasPrivilegedContainers := false
			for _, container := range pod.Containers {
				if container.SecurityContext != nil &&
					container.SecurityContext.Privileged != nil &&
					*container.SecurityContext.Privileged {

					containerReport := map[string]any{
						"name":       container.Name,
						"privileged": *container.SecurityContext.Privileged,
					}
					podReport.Containers = append(podReport.Containers, containerReport)
					hasPrivilegedContainers = true
				}
			}

			if hasPrivilegedContainers {
				nsReport.Pods = append(nsReport.Pods, podReport)
			}
		}

		if len(nsReport.Pods) > 0 {
			reportData = append(reportData, nsReport)
		}
	}

	report := &Report{
		Type:  "privileged",
		Count: g.countContainersInReport(reportData),
		Data:  reportData,
	}

	if err := g.outputReport(report); err != nil {
		return nil, err
	}

	return []*Report{report}, nil
}

// generatePrivilegeEscalationReport generates a report of containers allowing privilege escalation
func (g *Generator) generatePrivilegeEscalationReport() ([]*Report, error) {
	reportData := make([]*ReportNamespace, 0)

	for _, namespace := range g.data.Namespaces {
		nsReport := &ReportNamespace{
			Namespace: namespace.Name,
			Pods:      make([]any, 0),
		}

		for _, pod := range namespace.Pods {
			podReport := &PodReport{
				Name:       pod.Name,
				Containers: make([]any, 0),
			}

			hasIssues := false
			for _, container := range pod.Containers {
				if container.SecurityContext != nil &&
					container.SecurityContext.AllowPrivilegeEscalation != nil &&
					*container.SecurityContext.AllowPrivilegeEscalation {

					containerReport := map[string]any{
						"name":                     container.Name,
						"allowPrivilegeEscalation": *container.SecurityContext.AllowPrivilegeEscalation,
					}
					podReport.Containers = append(podReport.Containers, containerReport)
					hasIssues = true
				}
			}

			if hasIssues {
				nsReport.Pods = append(nsReport.Pods, podReport)
			}
		}

		if len(nsReport.Pods) > 0 {
			reportData = append(reportData, nsReport)
		}
	}

	report := &Report{
		Type:  "privesc",
		Count: g.countContainersInReport(reportData),
		Data:  reportData,
	}

	if err := g.outputReport(report); err != nil {
		return nil, err
	}

	return []*Report{report}, nil
}

// generateNonRootReport generates a report of containers not running as non-root
func (g *Generator) generateNonRootReport() ([]*Report, error) {
	reportData := make([]*ReportNamespace, 0)

	for _, namespace := range g.data.Namespaces {
		nsReport := &ReportNamespace{
			Namespace: namespace.Name,
			Pods:      make([]any, 0),
		}

		for _, pod := range namespace.Pods {
			podReport := &PodReport{
				Name:       pod.Name,
				Containers: make([]any, 0),
			}

			hasIssues := false
			for _, container := range pod.Containers {
				if container.SecurityContext == nil ||
					container.SecurityContext.RunAsNonRoot == nil ||
					!*container.SecurityContext.RunAsNonRoot {

					runAsNonRoot := false
					if container.SecurityContext != nil && container.SecurityContext.RunAsNonRoot != nil {
						runAsNonRoot = *container.SecurityContext.RunAsNonRoot
					}

					containerReport := map[string]any{
						"name":         container.Name,
						"runAsNonRoot": runAsNonRoot,
					}
					podReport.Containers = append(podReport.Containers, containerReport)
					hasIssues = true
				}
			}

			if hasIssues {
				nsReport.Pods = append(nsReport.Pods, podReport)
			}
		}

		if len(nsReport.Pods) > 0 {
			reportData = append(reportData, nsReport)
		}
	}

	report := &Report{
		Type:  "nonroot",
		Count: g.countContainersInReport(reportData),
		Data:  reportData,
	}

	if err := g.outputReport(report); err != nil {
		return nil, err
	}

	return []*Report{report}, nil
}

// generateCapabilitiesReport generates a report of dangerous capabilities
func (g *Generator) generateCapabilitiesReport() ([]*Report, error) {
	dangerousCaps := map[string]bool{
		"SYS_MODULE":      true,
		"NET_ADMIN":       true,
		"SYS_ADMIN":       true,
		"DAC_OVERRIDE":    true,
		"DAC_READ_SEARCH": true,
		"SYS_PTRACE":      true,
	}

	dropAllValues := map[string]bool{
		"ALL": true,
		"All": true,
		"all": true,
	}

	reportData := make([]*ReportNamespace, 0)

	for _, namespace := range g.data.Namespaces {
		nsReport := &ReportNamespace{
			Namespace: namespace.Name,
			Pods:      make([]any, 0),
		}

		for _, pod := range namespace.Pods {
			podReport := &PodReport{
				Name:       pod.Name,
				Containers: make([]any, 0),
			}

			hasIssues := false
			for _, container := range pod.Containers {
				if container.SecurityContext != nil && container.SecurityContext.Capabilities != nil {
					caps := container.SecurityContext.Capabilities
					hasRiskyCapabilities := false

					for _, addedCap := range caps.Add {
						if dangerousCaps[addedCap] {
							hasRiskyCapabilities = true
							break
						}
					}

					dropsAll := false
					for _, droppedCap := range caps.Drop {
						if dropAllValues[droppedCap] {
							dropsAll = true
							break
						}
					}

					if hasRiskyCapabilities || !dropsAll {
						containerReport := map[string]any{
							"name":    container.Name,
							"capAdd":  caps.Add,
							"capDrop": caps.Drop,
						}
						podReport.Containers = append(podReport.Containers, containerReport)
						hasIssues = true
					}
				}
			}

			if hasIssues {
				nsReport.Pods = append(nsReport.Pods, podReport)
			}
		}

		if len(nsReport.Pods) > 0 {
			reportData = append(reportData, nsReport)
		}
	}

	report := &Report{
		Type:  "caps",
		Count: g.countContainersInReport(reportData),
		Data:  reportData,
	}

	if err := g.outputReport(report); err != nil {
		return nil, err
	}

	return []*Report{report}, nil
}

// generateImageSourceReport generates a report of untrusted image sources
func (g *Generator) generateImageSourceReport() ([]*Report, error) {
	trustedRegistries := []string{"quay.io", "icr.io"}

	// Load trusted registries from file if provided
	if g.config.TrustedRegistries != "" {
		if content, err := os.ReadFile(g.config.TrustedRegistries); err == nil {
			lines := strings.Split(string(content), "\n")
			trustedRegistries = make([]string, 0)
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					trustedRegistries = append(trustedRegistries, line)
				}
			}
		}
	}

	reportData := make([]*ReportNamespace, 0)

	for _, namespace := range g.data.Namespaces {
		nsReport := &ReportNamespace{
			Namespace: namespace.Name,
			Pods:      make([]any, 0),
		}

		for _, pod := range namespace.Pods {
			podReport := &PodReport{
				Name:       pod.Name,
				Containers: make([]any, 0),
			}

			hasIssues := false
			for _, container := range pod.Containers {
				isUntrusted := true

				if strings.Contains(container.Image, ":latest") {
					isUntrusted = true
				} else {
					// Check if image is from trusted registry
					for _, registry := range trustedRegistries {
						if strings.Contains(container.Image, registry) {
							isUntrusted = false
							break
						}
					}
				}

				if isUntrusted {
					containerReport := map[string]any{
						"name":        container.Name,
						"imageSource": container.Image,
					}
					podReport.Containers = append(podReport.Containers, containerReport)
					hasIssues = true
				}
			}

			if hasIssues {
				nsReport.Pods = append(nsReport.Pods, podReport)
			}
		}

		if len(nsReport.Pods) > 0 {
			reportData = append(reportData, nsReport)
		}
	}

	report := &Report{
		Type:  "imgsrc",
		Count: g.countContainersInReport(reportData),
		Data:  reportData,
	}

	if err := g.outputReport(report); err != nil {
		return nil, err
	}

	return []*Report{report}, nil
}

// generateSeccompReport generates a report of pods without seccomp profiles
func (g *Generator) generateSeccompReport() ([]*Report, error) {
	reportData := make([]*ReportNamespace, 0)

	for _, namespace := range g.data.Namespaces {
		nsReport := &ReportNamespace{
			Namespace: namespace.Name,
			Pods:      make([]any, 0),
		}

		for _, pod := range namespace.Pods {
			// Check if pod has no seccomp profile or has "null" profile
			if pod.SecurityContext == nil ||
				pod.SecurityContext.SeccompProfile == "" ||
				pod.SecurityContext.SeccompProfile == "null" {

				seccompProfile := "null"
				if pod.SecurityContext != nil && pod.SecurityContext.SeccompProfile != "" {
					seccompProfile = pod.SecurityContext.SeccompProfile
				}

				podReport := map[string]any{
					"name":           pod.Name,
					"seccompProfile": seccompProfile,
				}
				nsReport.Pods = append(nsReport.Pods, podReport)
			}
		}

		if len(nsReport.Pods) > 0 {
			reportData = append(reportData, nsReport)
		}
	}

	report := &Report{
		Type:  "seccomp",
		Count: g.countPodsInReport(reportData),
		Data:  reportData,
	}

	if err := g.outputReport(report); err != nil {
		return nil, err
	}

	return []*Report{report}, nil
}

// generateLimitsReport generates a report of pods without resource limits
func (g *Generator) generateLimitsReport() ([]*Report, error) {
	reportData := make([]*ReportNamespace, 0)

	for _, namespace := range g.data.Namespaces {
		nsReport := &ReportNamespace{
			Namespace: namespace.Name,
			Pods:      make([]any, 0),
		}

		for _, pod := range namespace.Pods {
			// Check if any container lacks resource limits
			missingLimits := false
			for _, container := range pod.Containers {
				if container.CPULimit == "" || container.MemoryLimit == "" {
					missingLimits = true
					break
				}
			}

			if missingLimits {
				podReport := map[string]any{
					"name":           pod.Name,
					"resourceLimits": "null",
				}
				nsReport.Pods = append(nsReport.Pods, podReport)
			}
		}

		if len(nsReport.Pods) > 0 {
			reportData = append(reportData, nsReport)
		}
	}

	report := &Report{
		Type:  "limits",
		Count: g.countPodsInReport(reportData),
		Data:  reportData,
	}

	if err := g.outputReport(report); err != nil {
		return nil, err
	}

	return []*Report{report}, nil
}

// generateServiceAccountReport generates a report of service accounts
func (g *Generator) generateServiceAccountReport() ([]*Report, error) {
	reportData := make([]*ReportNamespace, 0)

	for _, namespace := range g.data.Namespaces {
		nsReport := &ReportNamespace{
			Namespace: namespace.Name,
			Pods:      make([]any, 0),
		}

		for _, pod := range namespace.Pods {
			podReport := map[string]any{
				"name":           pod.Name,
				"serviceAccount": pod.ServiceAccount,
			}
			nsReport.Pods = append(nsReport.Pods, podReport)
		}

		if len(nsReport.Pods) > 0 {
			reportData = append(reportData, nsReport)
		}
	}

	report := &Report{
		Type:  "serviceaccount",
		Count: g.countPodsInReport(reportData),
		Data:  reportData,
	}

	if err := g.outputReport(report); err != nil {
		return nil, err
	}

	return []*Report{report}, nil
}

// generateTokenReport generates a report of service account tokens
func (g *Generator) generateTokenReport() ([]*Report, error) {
	reportData := make([]*ReportNamespace, 0)

	for _, namespace := range g.data.Namespaces {
		serviceaccounts := make([]any, 0)
		for _, sa := range namespace.ServiceAccounts {
			if sa.Token != "" {
				saReport := map[string]any{
					"name":  sa.Name,
					"token": sa.Token,
				}
				serviceaccounts = append(serviceaccounts, saReport)
			}
		}

		if len(serviceaccounts) > 0 {
			nsData := map[string]any{
				"namespace":       namespace.Name,
				"serviceaccounts": serviceaccounts,
			}
			reportData = append(reportData, &ReportNamespace{
				Namespace: namespace.Name,
				Pods:      []interface{}{nsData},
			})
		}
	}

	report := &Report{
		Type:  "token",
		Count: g.countTokensInReport(reportData),
		Data:  reportData,
	}

	if err := g.outputReport(report); err != nil {
		return nil, err
	}

	return []*Report{report}, nil
}

// generateStatsReport generates a resource type inventory from the collection
func (g *Generator) generateStatsReport() ([]*Report, error) {
	entries := make([]StatsEntry, 0, len(g.data.ResourceCounts))
	total := 0
	for key, count := range g.data.ResourceCounts {
		parts := strings.SplitN(key, "|", 2)
		entries = append(entries, StatsEntry{APIVersion: parts[0], Kind: parts[1], Count: count})
		total += count
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].APIVersion != entries[j].APIVersion {
			if entries[i].APIVersion == "v1" {
				return true
			}
			if entries[j].APIVersion == "v1" {
				return false
			}
			return entries[i].APIVersion < entries[j].APIVersion
		}
		return entries[i].Kind < entries[j].Kind
	})

	report := &Report{Type: "stats", Count: total, Data: entries}
	if err := g.outputReport(report); err != nil {
		return nil, err
	}
	return []*Report{report}, nil
}

// generateAllReports generates all report types
func (g *Generator) generateAllReports() ([]*Report, error) {
	var allReports []*Report

	// Generate all security report types
	reportTypes := []string{"privileged", "privesc", "nonroot", "caps", "imgsrc", "seccomp", "limits", "serviceaccount", "token", "stats"}

	for _, reportType := range reportTypes {
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
		case "stats":
			reports, err = g.generateStatsReport()
		}

		if err != nil {
			return nil, fmt.Errorf("failed to generate %s report: %w", reportType, err)
		}

		allReports = append(allReports, reports...)
	}

	return allReports, nil
}

// countContainersInReport counts containers in a container-based report
func (g *Generator) countContainersInReport(reportData []*ReportNamespace) int {
	count := 0
	for _, ns := range reportData {
		for _, podInterface := range ns.Pods {
			if pod, ok := podInterface.(*PodReport); ok {
				count += len(pod.Containers)
			}
		}
	}
	return count
}

// countPodsInReport counts pods in a pod-based report
func (g *Generator) countPodsInReport(reportData []*ReportNamespace) int {
	count := 0
	for _, ns := range reportData {
		count += len(ns.Pods)
	}
	return count
}

// countTokensInReport counts tokens in a token report
func (g *Generator) countTokensInReport(reportData []*ReportNamespace) int {
	count := 0
	for _, ns := range reportData {
		for _, podInterface := range ns.Pods {
			if nsData, ok := podInterface.(map[string]any); ok {
				if sas, ok := nsData["serviceaccounts"].([]any); ok {
					count += len(sas)
				}
			}
		}
	}
	return count
}

// outputReport outputs the report to stdout or file
func (g *Generator) outputReport(report *Report) error {
	switch g.config.Format {
	case "json":
		jsonData, err := json.MarshalIndent(report.Data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}

		// Output to file or stdout
		if g.config.OutputPrefix != "" {
			filename := fmt.Sprintf("%s_%s.json", g.config.OutputPrefix, report.Type)
			if err := os.WriteFile(filename, jsonData, 0644); err != nil {
				return fmt.Errorf("failed to write report file: %w", err)
			}
		} else {
			fmt.Println(string(jsonData))
		}
	case "csv":
		// Generate CSV data
		csvData, err := g.generateCSVData(report)
		if err != nil {
			return fmt.Errorf("failed to generate CSV: %w", err)
		}

		// Write CSV data
		if err := g.writeCSV(csvData, report); err != nil {
			return fmt.Errorf("failed to write CSV: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", g.config.Format)
	}

	return nil
}
