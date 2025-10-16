package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
)

// generateCSVData converts report data to CSV format
func (g *Generator) generateCSVData(report *Report) ([][]string, error) {
	switch report.Type {
	case "token":
		return g.generateTokenCSV(report.Data)
	default:
		// Handle container-based and pod-based reports
		return g.generateStandardCSV(report.Data, report.Type)
	}
}

// generateTokenCSV generates CSV for token reports with specific format
func (g *Generator) generateTokenCSV(data any) ([][]string, error) {
	reportData, ok := data.([]*ReportNamespace)
	if !ok {
		return nil, fmt.Errorf("invalid data format for token CSV")
	}

	// Headers: Namespace, Name, Token
	headers := []string{"Namespace", "Name", "Token"}
	var rows [][]string
	rows = append(rows, headers)

	for _, ns := range reportData {
		for _, podInterface := range ns.Pods {
			if nsData, ok := podInterface.(map[string]any); ok {
				if sas, ok := nsData["serviceaccounts"].([]any); ok {
					for _, saInterface := range sas {
						if sa, ok := saInterface.(map[string]any); ok {
							name, _ := sa["name"].(string)
							token, _ := sa["token"].(string)
							row := []string{ns.Namespace, name, token}
							rows = append(rows, row)
						}
					}
				}
			}
		}
	}

	return rows, nil
}

// generateStandardCSV generates CSV for standard reports (container-based and pod-based)
func (g *Generator) generateStandardCSV(data any, reportType string) ([][]string, error) {
	reportData, ok := data.([]*ReportNamespace)
	if !ok {
		return nil, fmt.Errorf("invalid data format for CSV")
	}

	var rows [][]string
	var headers []string

	// Determine if this is a container-based report
	isContainerBased := g.isContainerBasedReport(reportType)

	if isContainerBased {
		// For container-based reports, we need to flatten the nested structure
		// Headers will be: Namespace, Pods_Name, Name, [container-specific fields]
		headers = []string{"Namespace", "Pods_Name", "Name"}

		// Collect all unique field names from containers to build complete headers
		fieldSet := make(map[string]bool)
		for _, ns := range reportData {
			for _, podInterface := range ns.Pods {
				if pod, ok := podInterface.(*PodReport); ok {
					for _, containerInterface := range pod.Containers {
						if container, ok := containerInterface.(map[string]interface{}); ok {
							for field := range container {
								if field != "name" { // name is already in headers
									fieldSet[field] = true
								}
							}
						}
					}
				}
			}
		}

		// Sort fields for consistent ordering
		var fields []string
		for field := range fieldSet {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		headers = append(headers, fields...)

		// Reorder headers according to Python reference patterns
		headers = g.reorderContainerHeaders(headers, reportType)
		rows = append(rows, headers)

		// Generate data rows
		for _, ns := range reportData {
			for _, podInterface := range ns.Pods {
				if pod, ok := podInterface.(*PodReport); ok {
					for _, containerInterface := range pod.Containers {
						if container, ok := containerInterface.(map[string]interface{}); ok {
							row := make([]string, len(headers))
							row[0] = ns.Namespace
							row[1] = pod.Name

							name, _ := container["name"].(string)
							row[2] = name

							// Fill in the remaining fields
							for i, header := range headers[3:] {
								if value, exists := container[header]; exists {
									row[i+3] = g.formatValue(value)
								}
							}
							rows = append(rows, row)
						}
					}
				}
			}
		}
	} else {
		// For pod-based reports
		// Headers will be: Namespace, Name, [pod-specific fields]
		headers = []string{"Namespace", "Name"}

		// Collect all unique field names
		fieldSet := make(map[string]bool)
		for _, ns := range reportData {
			for _, podInterface := range ns.Pods {
				if pod, ok := podInterface.(map[string]interface{}); ok {
					for field := range pod {
						if field != "name" { // name is already in headers
							fieldSet[field] = true
						}
					}
				}
			}
		}

		// Sort fields for consistent ordering
		var fields []string
		for field := range fieldSet {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		headers = append(headers, fields...)

		// Reorder headers according to Python reference patterns
		headers = g.reorderPodHeaders(headers, reportType)
		rows = append(rows, headers)

		// Generate data rows
		for _, ns := range reportData {
			for _, podInterface := range ns.Pods {
				if pod, ok := podInterface.(map[string]any); ok {
					row := make([]string, len(headers))
					row[0] = ns.Namespace

					name, _ := pod["name"].(string)
					row[1] = name

					// Fill in the remaining fields
					for i, header := range headers[2:] {
						if value, exists := pod[header]; exists {
							row[i+2] = g.formatValue(value)
						}
					}
					rows = append(rows, row)
				}
			}
		}
	}

	return rows, nil
}

// isContainerBasedReport determines if a report type is container-based
func (g *Generator) isContainerBasedReport(reportType string) bool {
	containerBasedReports := map[string]bool{
		"privileged": true,
		"privesc":    true,
		"nonroot":    true,
		"caps":       true,
		"imgsrc":     true,
	}
	return containerBasedReports[reportType]
}

// reorderContainerHeaders reorders headers based on Python reference patterns
func (g *Generator) reorderContainerHeaders(headers []string, reportType string) []string {
	// Based on Python reference, the order should be optimized for readability
	switch reportType {
	case "privileged":
		return g.moveFieldsToEnd(headers, []string{"privileged"})
	case "privesc":
		return g.moveFieldsToEnd(headers, []string{"allowPrivilegeEscalation"})
	case "nonroot":
		return g.moveFieldsToEnd(headers, []string{"runAsNonRoot"})
	case "caps":
		return g.moveFieldsToEnd(headers, []string{"capAdd", "capDrop"})
	case "imgsrc":
		return g.moveFieldsToEnd(headers, []string{"imageSource"})
	default:
		return headers
	}
}

// reorderPodHeaders reorders headers for pod-based reports
func (g *Generator) reorderPodHeaders(headers []string, reportType string) []string {
	switch reportType {
	case "seccomp":
		return g.moveFieldsToEnd(headers, []string{"seccompProfile"})
	case "limits":
		return g.moveFieldsToEnd(headers, []string{"resourceLimits"})
	case "serviceaccount":
		return g.moveFieldsToEnd(headers, []string{"serviceAccount"})
	default:
		return headers
	}
}

// moveFieldsToEnd moves specified fields to the end of the header list
func (g *Generator) moveFieldsToEnd(headers []string, fields []string) []string {
	var result []string
	var endFields []string

	fieldSet := make(map[string]bool)
	for _, field := range fields {
		fieldSet[field] = true
	}

	// Add non-target fields first
	for _, header := range headers {
		if fieldSet[header] {
			endFields = append(endFields, header)
		} else {
			result = append(result, header)
		}
	}

	// Add target fields at the end
	result = append(result, endFields...)
	return result
}

// formatValue converts various data types to string for CSV
func (g *Generator) formatValue(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case map[string]any:
		// For complex objects like capabilities, create a more readable format
		if add, hasAdd := v["add"]; hasAdd {
			if drop, hasDrop := v["drop"]; hasDrop {
				return fmt.Sprintf("add:%s,drop:%s", g.formatValue(add), g.formatValue(drop))
			}
			return fmt.Sprintf("add:%s", g.formatValue(add))
		}
		if drop, hasDrop := v["drop"]; hasDrop {
			return fmt.Sprintf("drop:%s", g.formatValue(drop))
		}
		// Fallback to generic format
		var parts []string
		for k, val := range v {
			parts = append(parts, fmt.Sprintf("%s:%v", k, val))
		}
		return fmt.Sprintf("{%s}", strings.Join(parts, ","))
	case []interface{}:
		// For arrays, join with semicolons
		var items []string
		for _, item := range v {
			items = append(items, g.formatValue(item))
		}
		return strings.Join(items, ";")
	default:
		// Use reflection for other types
		return fmt.Sprintf("%v", value)
	}
}

// writeCSV writes CSV data to file or stdout
func (g *Generator) writeCSV(rows [][]string, report *Report) error {
	if g.config.OutputPrefix != "" {
		filename := fmt.Sprintf("%s_%s.csv", g.config.OutputPrefix, report.Type)
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		for _, row := range rows {
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
		}
		return nil
	} else {
		// Write to stdout
		writer := csv.NewWriter(os.Stdout)
		defer writer.Flush()

		for _, row := range rows {
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
		}
		return nil
	}
}
