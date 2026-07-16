package collector

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type AllowlistEntry struct {
	Group    string
	Version  string
	Resource string
}

var defaultAllowlist = []string{
	"v1/secrets",
	"v1/configmaps",
	"v1/pods",
	"v1/services",
	"v1/nodes",
	"v1/persistentvolumes",
	"v1/persistentvolumeclaims",
	"v1/serviceaccounts",
	"apps/v1/deployments",
	"apps/v1/daemonsets",
	"apps/v1/statefulsets",
	"batch/v1/cronjobs",
	"batch/v1/jobs",
	"networking.k8s.io/v1/ingresses",
	"networking.k8s.io/v1/networkpolicies",
	"projectcalico.org/v3/globalnetworkpolicies",
	"cilium.io/v2/ciliumnetworkpolicies",
	"gateway.networking.k8s.io/v1",
	"gateway.networking.k8s.io/v1alpha2/grpcroutes",
	"gateway.networking.k8s.io/v1alpha2/tcproutes",
	"gateway.networking.k8s.io/v1alpha2/tlsroutes",
	"gateway.networking.k8s.io/v1beta1/gateways",
	"apiextensions.k8s.io/v1/customresourcedefinitions",
	"rbac.authorization.k8s.io/v1/roles",
	"rbac.authorization.k8s.io/v1/clusterroles",
	"rbac.authorization.k8s.io/v1/rolebindings",
	"rbac.authorization.k8s.io/v1/clusterrolebindings",
	"route.openshift.io/v1/routes",
	"project.openshift.io/v1/projects",
	"image.openshift.io/v1/imagestreams",
}

func DefaultDiscoveryAllowlist() ([]AllowlistEntry, error) {
	return ParseAllowlistEntries(defaultAllowlist)
}

func MergeAllowlists(base []AllowlistEntry, extra []AllowlistEntry) []AllowlistEntry {
	if len(base) == 0 {
		return extra
	}
	if len(extra) == 0 {
		return base
	}

	seen := make(map[string]struct{}, len(base)+len(extra))
	merged := make([]AllowlistEntry, 0, len(base)+len(extra))

	add := func(entry AllowlistEntry) {
		key := fmt.Sprintf("%s|%s|%s", entry.Group, entry.Version, entry.Resource)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		merged = append(merged, entry)
	}

	for _, entry := range base {
		add(entry)
	}
	for _, entry := range extra {
		add(entry)
	}

	return merged
}

func ParseAllowlistFile(path string) ([]AllowlistEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entries = append(entries, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ParseAllowlistEntries(entries)
}

func ParseAllowlistEntries(entries []string) ([]AllowlistEntry, error) {
	parsed := make([]AllowlistEntry, 0, len(entries))
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}

		parts := strings.Split(trimmed, "/")
		switch len(parts) {
		case 1:
			if strings.Contains(parts[0], ".") {
				group := parts[0]
				if group == "core" {
					group = ""
				}
				parsed = append(parsed, AllowlistEntry{Group: group})
			} else {
				parsed = append(parsed, AllowlistEntry{Resource: parts[0]})
			}
		case 2:
			if isAPIVersion(parts[0]) {
				parsed = append(parsed, AllowlistEntry{Version: parts[0], Resource: parts[1]})
			} else {
				group := parts[0]
				if group == "core" {
					group = ""
				}
				if parts[1] == "*" {
					parsed = append(parsed, AllowlistEntry{Group: group})
				} else if isAPIVersion(parts[1]) {
					parsed = append(parsed, AllowlistEntry{Group: group, Version: parts[1]})
				} else {
					parsed = append(parsed, AllowlistEntry{Group: group, Resource: parts[1]})
				}
			}
		case 3:
			group := parts[0]
			if group == "core" {
				group = ""
			}
			resource := parts[2]
			if resource == "*" {
				resource = ""
			}
			parsed = append(parsed, AllowlistEntry{Group: group, Version: parts[1], Resource: resource})
		default:
			return nil, fmt.Errorf("invalid allowlist entry %q", entry)
		}
	}

	return parsed, nil
}

func FilterDiscoveredResources(resources []DiscoveryResource, allowlist []AllowlistEntry) []DiscoveryResource {
	if len(allowlist) == 0 {
		return nil
	}

	filtered := make([]DiscoveryResource, 0, len(resources))
	for _, res := range resources {
		for _, entry := range allowlist {
			if !matchesAllowlist(res, entry) {
				continue
			}
			filtered = append(filtered, res)
			break
		}
	}

	return filtered
}

func matchesAllowlist(resource DiscoveryResource, entry AllowlistEntry) bool {
	if entry.Group != "" && resource.Group != entry.Group {
		return false
	}
	if entry.Version != "" && resource.Version != entry.Version {
		return false
	}
	if entry.Resource != "" && resource.Resource != entry.Resource {
		return false
	}
	return true
}

func isAPIVersion(value string) bool {
	if len(value) < 2 || value[0] != 'v' {
		return false
	}
	return value[1] >= '0' && value[1] <= '9'
}
