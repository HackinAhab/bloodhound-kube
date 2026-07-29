//go:build !no_addons && !no_istio

package istio

import (
	"strconv"
	"strings"

	. "bloodhound-kube/internal/nodes/framework"
)

func Register() {
	RegisterKind("Gateway", BuildIstioGatewayNode)
	RegisterKind("VirtualService", BuildVirtualServiceNode)
	RegisterKind("PeerAuthentication", BuildPeerAuthenticationNode)
	RegisterKind("AuthorizationPolicy", BuildAuthorizationPolicyNode)
}

func BuildIstioGatewayNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	selectorLabels := stringMapFromAny(GetMap(spec, "selector"))
	secretRefs := extractGatewaySecretRefs(GetSlice(spec, "servers"), namespace)

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"selector":    MapToSortedList(GetMap(spec, "selector")),
		"servers":     summarizeGatewayServers(GetSlice(spec, "servers")),
	}

	base := NewGraphNodeBase("BHK_IstioGateway", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: IstioGateway{
			GraphNodeBase:  base,
			SelectorLabels: selectorLabels,
			SecretRefs:     secretRefs,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func BuildVirtualServiceNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	gatewayRefs := extractVirtualServiceGatewayRefs(GetSlice(spec, "gateways"), namespace)
	destinationHosts := extractVirtualServiceDestinationHosts(spec)

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"hosts":       StringSliceFromAny(GetSlice(spec, "hosts")),
		"gateways":    StringSliceFromAny(GetSlice(spec, "gateways")),
	}

	base := NewGraphNodeBase("BHK_VirtualService", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: VirtualService{
			GraphNodeBase:    base,
			GatewayRefs:      gatewayRefs,
			DestinationHosts: destinationHosts,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func BuildPeerAuthenticationNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	selectorLabels := stringMapFromAny(GetMap(GetMap(spec, "selector"), "matchLabels"))
	mtlsMode := GetStringDefault(GetMap(spec, "mtls"), "mode", "UNSET")

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"mtlsMode":    mtlsMode,
		"permissive":  mtlsMode == "PERMISSIVE",
	}

	base := NewGraphNodeBase("BHK_PeerAuthentication", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: PeerAuthentication{
			GraphNodeBase:  base,
			SelectorLabels: selectorLabels,
			MTLSMode:       mtlsMode,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func BuildAuthorizationPolicyNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	selectorLabels := stringMapFromAny(GetMap(GetMap(spec, "selector"), "matchLabels"))
	action := GetStringDefault(spec, "action", "ALLOW")
	rules := GetSlice(spec, "rules")
	principals := extractAuthorizationPolicyPrincipals(rules)
	allowAll := action == "ALLOW" && len(rules) == 0

	properties := map[string]any{
		"name":        name,
		"namespace":   namespace,
		"labels":      MapToSortedList(labelsMap),
		"annotations": MapToSortedList(annotationsMap),
		"action":      action,
		"allowAll":    allowAll,
		"principals":  principalStrings(principals),
	}

	base := NewGraphNodeBase("BHK_AuthorizationPolicy", namespace, name, labelsMap, annotationsMap)

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: AuthorizationPolicy{
			GraphNodeBase:  base,
			SelectorLabels: selectorLabels,
			Action:         action,
			Principals:     principals,
		},
	}

	return BuildResult{
		Node: NewNodeResult(base, properties),
		Core: []CoreEntry{core},
	}, true
}

func stringMapFromAny(m map[string]any) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

func summarizeGatewayServers(servers []any) []string {
	if len(servers) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(servers))
	for _, item := range servers {
		server, ok := item.(map[string]any)
		if !ok {
			continue
		}
		port := GetMap(server, "port")
		entry := GetString(port, "protocol") + ":" + formatPortNumber(port)
		if tls := GetMap(server, "tls"); len(tls) > 0 {
			if cred := GetString(tls, "credentialName"); cred != "" {
				entry += " credentialName=" + cred
			}
		}
		items = append(items, entry)
	}
	return items
}

func formatPortNumber(port map[string]any) string {
	switch v := port["number"].(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		return "0"
	}
}

// extractGatewaySecretRefs collects servers[].tls.credentialName references.
// Per Istio convention, credentialName resolves to a Secret in the same
// namespace as the Gateway resource (cross-namespace/istio-system credential
// resolution via Secret Discovery Service is intentionally out of scope).
func extractGatewaySecretRefs(servers []any, gatewayNamespace string) []SecretRef {
	if len(servers) == 0 {
		return []SecretRef{}
	}
	seen := map[string]struct{}{}
	var refs []SecretRef
	for _, item := range servers {
		server, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cred := GetString(GetMap(server, "tls"), "credentialName")
		if cred == "" {
			continue
		}
		if _, ok := seen[cred]; ok {
			continue
		}
		seen[cred] = struct{}{}
		refs = append(refs, SecretRef{Namespace: gatewayNamespace, Name: cred})
	}
	if refs == nil {
		return []SecretRef{}
	}
	return refs
}

// extractVirtualServiceGatewayRefs parses spec.gateways entries. Each entry
// is either a bare name (resolved in the VirtualService's own namespace) or
// "<namespace>/<name>"; the special value "mesh" (sidecar-to-sidecar, no
// Gateway resource) is skipped.
func extractVirtualServiceGatewayRefs(gateways []any, defaultNamespace string) []SecretRef {
	if len(gateways) == 0 {
		return []SecretRef{}
	}
	var refs []SecretRef
	for _, item := range gateways {
		name, ok := item.(string)
		if !ok || name == "" || name == "mesh" {
			continue
		}
		ns := defaultNamespace
		if idx := strings.Index(name, "/"); idx >= 0 {
			ns = name[:idx]
			name = name[idx+1:]
		}
		if name == "" {
			continue
		}
		refs = append(refs, SecretRef{Namespace: ns, Name: name})
	}
	if refs == nil {
		return []SecretRef{}
	}
	return refs
}

// extractVirtualServiceDestinationHosts collects route destination.host
// values across http/tls/tcp route rules.
func extractVirtualServiceDestinationHosts(spec map[string]any) []string {
	seen := map[string]struct{}{}
	collect := func(rules []any) {
		for _, item := range rules {
			rule, ok := item.(map[string]any)
			if !ok {
				continue
			}
			for _, key := range []string{"route"} {
				for _, r := range GetSlice(rule, key) {
					routeEntry, ok := r.(map[string]any)
					if !ok {
						continue
					}
					host := GetString(GetMap(routeEntry, "destination"), "host")
					if host != "" {
						seen[host] = struct{}{}
					}
				}
			}
		}
	}
	collect(GetSlice(spec, "http"))
	collect(GetSlice(spec, "tls"))
	collect(GetSlice(spec, "tcp"))
	return SortedSetKeys(seen)
}

// extractAuthorizationPolicyPrincipals parses SPIFFE-style principal strings
// ("cluster.local/ns/<namespace>/sa/<name>") from rules[].from[].source.principals.
func extractAuthorizationPolicyPrincipals(rules []any) []Principal {
	seen := map[string]struct{}{}
	var principals []Principal
	for _, r := range rules {
		rule, ok := r.(map[string]any)
		if !ok {
			continue
		}
		for _, f := range GetSlice(rule, "from") {
			from, ok := f.(map[string]any)
			if !ok {
				continue
			}
			source := GetMap(from, "source")
			for _, p := range GetSlice(source, "principals") {
				spiffe, ok := p.(string)
				if !ok {
					continue
				}
				ns, name, ok := parseSpiffePrincipal(spiffe)
				if !ok {
					continue
				}
				key := ns + "/" + name
				if _, dup := seen[key]; dup {
					continue
				}
				seen[key] = struct{}{}
				principals = append(principals, Principal{Namespace: ns, Name: name})
			}
		}
	}
	if principals == nil {
		return []Principal{}
	}
	return principals
}

// parseSpiffePrincipal parses "cluster.local/ns/<namespace>/sa/<name>" (and
// the equivalent form without a trust-domain prefix, "ns/<namespace>/sa/<name>").
func parseSpiffePrincipal(spiffe string) (namespace, name string, ok bool) {
	idx := strings.Index(spiffe, "/ns/")
	if idx < 0 {
		return "", "", false
	}
	rest := spiffe[idx+len("/ns/"):]
	parts := strings.SplitN(rest, "/sa/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func principalStrings(principals []Principal) []string {
	if len(principals) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(principals))
	for _, p := range principals {
		items = append(items, p.Namespace+"/"+p.Name)
	}
	return items
}
