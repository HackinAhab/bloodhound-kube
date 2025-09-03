package collector

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Route struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Kind        string            `json:"kind"` // HTTPRoute or GRPCRoute
	ParentRefs  []RouteParentRef  `json:"parent_refs,omitempty"`
	Hostnames   []string          `json:"hostnames,omitempty"`
	Rules       []RouteRule       `json:"rules,omitempty"`
	Status      RouteStatus       `json:"status,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

type RouteParentRef struct {
	Group       string `json:"group,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	SectionName string `json:"section_name,omitempty"`
	Port        int32  `json:"port,omitempty"`
}

type RouteRule struct {
	Matches     []RouteMatch      `json:"matches,omitempty"`
	Filters     []RouteFilter     `json:"filters,omitempty"`
	BackendRefs []RouteBackendRef `json:"backend_refs,omitempty"`
}

type RouteMatch struct {
	Path    *RoutePathMatch    `json:"path,omitempty"`
	Headers []RouteHeaderMatch `json:"headers,omitempty"`
	Query   []RouteQueryMatch  `json:"query,omitempty"`
	Method  string             `json:"method,omitempty"`
}

type RoutePathMatch struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type RouteHeaderMatch struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type RouteQueryMatch struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type RouteFilter struct {
	Type            string                       `json:"type"`
	RequestRedirect *RouteRequestRedirect        `json:"request_redirect,omitempty"`
	RequestRewrite  *RouteRequestHeaderModifier  `json:"request_rewrite,omitempty"`
	ResponseRewrite *RouteResponseHeaderModifier `json:"response_rewrite,omitempty"`
}

type RouteRequestRedirect struct {
	Scheme     string          `json:"scheme,omitempty"`
	Hostname   string          `json:"hostname,omitempty"`
	Path       *RoutePathMatch `json:"path,omitempty"`
	Port       int32           `json:"port,omitempty"`
	StatusCode int             `json:"status_code,omitempty"`
}

type RouteRequestHeaderModifier struct {
	Set    map[string]string `json:"set,omitempty"`
	Add    map[string]string `json:"add,omitempty"`
	Remove []string          `json:"remove,omitempty"`
}

type RouteResponseHeaderModifier struct {
	Set    map[string]string `json:"set,omitempty"`
	Add    map[string]string `json:"add,omitempty"`
	Remove []string          `json:"remove,omitempty"`
}

type RouteBackendRef struct {
	Group     string `json:"group,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Port      int32  `json:"port,omitempty"`
	Weight    int32  `json:"weight,omitempty"`
}

type RouteStatus struct {
	Parents []RouteParentStatus `json:"parents,omitempty"`
}

type RouteParentStatus struct {
	ParentRef      RouteParentRef   `json:"parent_ref"`
	ControllerName string           `json:"controller_name,omitempty"`
	Conditions     []RouteCondition `json:"conditions,omitempty"`
}

type RouteCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (c *Collector) CollectRoutes(ctx context.Context, namespace string) ([]Route, error) {
	c.logger.Info("Collecting routes (HTTPRoutes and GRPCRoutes)", "namespace", namespace)

	var allRoutes []Route

	// Collect HTTPRoutes
	httpRoutes, err := c.collectRoutesByKind(ctx, namespace, "HTTPRoute", "httproutes")
	if err != nil {
		c.logger.Debug("Failed to collect HTTPRoutes", "error", err)
	} else {
		allRoutes = append(allRoutes, httpRoutes...)
	}

	// Collect GRPCRoutes
	grpcRoutes, err := c.collectRoutesByKind(ctx, namespace, "GRPCRoute", "grpcroutes")
	if err != nil {
		c.logger.Debug("Failed to collect GRPCRoutes", "error", err)
	} else {
		allRoutes = append(allRoutes, grpcRoutes...)
	}

	c.logger.Info("Successfully collected routes", "count", len(allRoutes))
	return allRoutes, nil
}

func (c *Collector) collectRoutesByKind(ctx context.Context, namespace, kind, resource string) ([]Route, error) {
	routeGVR := schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: resource,
	}

	dynamicClient := c.clients.Kubernetes.Discovery().RESTClient()
	result := dynamicClient.Get().
		AbsPath("/apis", routeGVR.Group, routeGVR.Version, "namespaces", namespace, routeGVR.Resource).
		Do(ctx)

	rawData, err := result.Raw()
	if err != nil {
		c.logger.Debug("Gateway API routes not available or no routes found", "kind", kind, "error", err)
		return []Route{}, nil
	}

	routeList := &unstructured.UnstructuredList{}
	if err := routeList.UnmarshalJSON(rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s list: %w", kind, err)
	}

	routes := make([]Route, 0, len(routeList.Items))
	for _, item := range routeList.Items {
		route := Route{
			Name:        item.GetName(),
			Namespace:   item.GetNamespace(),
			Kind:        kind,
			Labels:      item.GetLabels(),
			Annotations: item.GetAnnotations(),
		}

		if creationTime := item.GetCreationTimestamp(); !creationTime.IsZero() {
			route.CreatedAt = creationTime.Format("2006-01-02T15:04:05Z")
		}

		if spec, found, _ := unstructured.NestedMap(item.Object, "spec"); found {
			// Parse ParentRefs
			if parentRefsRaw, found, _ := unstructured.NestedSlice(spec, "parentRefs"); found {
				route.ParentRefs = parseParentRefs(parentRefsRaw)
			}

			// Parse Hostnames
			if hostnamesRaw, found, _ := unstructured.NestedStringSlice(spec, "hostnames"); found {
				route.Hostnames = hostnamesRaw
			}

			// Parse Rules
			if rulesRaw, found, _ := unstructured.NestedSlice(spec, "rules"); found {
				route.Rules = parseRouteRules(rulesRaw, kind)
			}
		}

		// Parse Status
		if status, found, _ := unstructured.NestedMap(item.Object, "status"); found {
			route.Status = parseRouteStatus(status)
		}

		routes = append(routes, route)
	}

	return routes, nil
}

func parseParentRefs(parentRefsRaw []any) []RouteParentRef {
	var parentRefs []RouteParentRef
	for _, parentRefRaw := range parentRefsRaw {
		if parentRefMap, ok := parentRefRaw.(map[string]any); ok {
			parentRef := RouteParentRef{}
			if group, found, _ := unstructured.NestedString(parentRefMap, "group"); found {
				parentRef.Group = group
			}
			if kind, found, _ := unstructured.NestedString(parentRefMap, "kind"); found {
				parentRef.Kind = kind
			}
			if name, found, _ := unstructured.NestedString(parentRefMap, "name"); found {
				parentRef.Name = name
			}
			if namespace, found, _ := unstructured.NestedString(parentRefMap, "namespace"); found {
				parentRef.Namespace = namespace
			}
			if sectionName, found, _ := unstructured.NestedString(parentRefMap, "sectionName"); found {
				parentRef.SectionName = sectionName
			}
			if port, found, _ := unstructured.NestedInt64(parentRefMap, "port"); found {
				parentRef.Port = int32(port)
			}
			parentRefs = append(parentRefs, parentRef)
		}
	}
	return parentRefs
}

func parseRouteRules(rulesRaw []any, routeKind string) []RouteRule {
	var rules []RouteRule
	for _, ruleRaw := range rulesRaw {
		if ruleMap, ok := ruleRaw.(map[string]any); ok {
			rule := RouteRule{}

			// Parse Matches
			if matchesRaw, found, _ := unstructured.NestedSlice(ruleMap, "matches"); found {
				rule.Matches = parseRouteMatches(matchesRaw, routeKind)
			}

			// Parse Filters
			if filtersRaw, found, _ := unstructured.NestedSlice(ruleMap, "filters"); found {
				rule.Filters = parseRouteFilters(filtersRaw)
			}

			// Parse BackendRefs
			if backendRefsRaw, found, _ := unstructured.NestedSlice(ruleMap, "backendRefs"); found {
				rule.BackendRefs = parseBackendRefs(backendRefsRaw)
			}

			rules = append(rules, rule)
		}
	}
	return rules
}

func parseRouteMatches(matchesRaw []any, routeKind string) []RouteMatch {
	var matches []RouteMatch
	for _, matchRaw := range matchesRaw {
		if matchMap, ok := matchRaw.(map[string]any); ok {
			match := RouteMatch{}

			if routeKind == "HTTPRoute" {
				if pathMap, found, _ := unstructured.NestedMap(matchMap, "path"); found {
					pathMatch := &RoutePathMatch{}
					if pathType, found, _ := unstructured.NestedString(pathMap, "type"); found {
						pathMatch.Type = pathType
					}
					if value, found, _ := unstructured.NestedString(pathMap, "value"); found {
						pathMatch.Value = value
					}
					match.Path = pathMatch
				}

				if method, found, _ := unstructured.NestedString(matchMap, "method"); found {
					match.Method = method
				}
			}

			if headersRaw, found, _ := unstructured.NestedSlice(matchMap, "headers"); found {
				var headers []RouteHeaderMatch
				for _, headerRaw := range headersRaw {
					if headerMap, ok := headerRaw.(map[string]any); ok {
						header := RouteHeaderMatch{}
						if headerType, found, _ := unstructured.NestedString(headerMap, "type"); found {
							header.Type = headerType
						}
						if name, found, _ := unstructured.NestedString(headerMap, "name"); found {
							header.Name = name
						}
						if value, found, _ := unstructured.NestedString(headerMap, "value"); found {
							header.Value = value
						}
						headers = append(headers, header)
					}
				}
				match.Headers = headers
			}

			if routeKind == "HTTPRoute" {
				if queryParamsRaw, found, _ := unstructured.NestedSlice(matchMap, "queryParams"); found {
					var queryParams []RouteQueryMatch
					for _, queryRaw := range queryParamsRaw {
						if queryMap, ok := queryRaw.(map[string]any); ok {
							query := RouteQueryMatch{}
							if queryType, found, _ := unstructured.NestedString(queryMap, "type"); found {
								query.Type = queryType
							}
							if name, found, _ := unstructured.NestedString(queryMap, "name"); found {
								query.Name = name
							}
							if value, found, _ := unstructured.NestedString(queryMap, "value"); found {
								query.Value = value
							}
							queryParams = append(queryParams, query)
						}
					}
					match.Query = queryParams
				}
			}

			matches = append(matches, match)
		}
	}
	return matches
}

func parseRouteFilters(filtersRaw []any) []RouteFilter {
	var filters []RouteFilter
	for _, filterRaw := range filtersRaw {
		if filterMap, ok := filterRaw.(map[string]any); ok {
			filter := RouteFilter{}
			if filterType, found, _ := unstructured.NestedString(filterMap, "type"); found {
				filter.Type = filterType
			}

			switch filter.Type {
			case "RequestRedirect":
				if redirectMap, found, _ := unstructured.NestedMap(filterMap, "requestRedirect"); found {
					redirect := &RouteRequestRedirect{}
					if scheme, found, _ := unstructured.NestedString(redirectMap, "scheme"); found {
						redirect.Scheme = scheme
					}
					if hostname, found, _ := unstructured.NestedString(redirectMap, "hostname"); found {
						redirect.Hostname = hostname
					}
					if port, found, _ := unstructured.NestedInt64(redirectMap, "port"); found {
						redirect.Port = int32(port)
					}
					if statusCode, found, _ := unstructured.NestedInt64(redirectMap, "statusCode"); found {
						redirect.StatusCode = int(statusCode)
					}
					filter.RequestRedirect = redirect
				}
			case "RequestHeaderModifier":
				if modifierMap, found, _ := unstructured.NestedMap(filterMap, "requestHeaderModifier"); found {
					modifier := &RouteRequestHeaderModifier{}
					if setMap, found, _ := unstructured.NestedStringMap(modifierMap, "set"); found {
						modifier.Set = setMap
					}
					if addMap, found, _ := unstructured.NestedStringMap(modifierMap, "add"); found {
						modifier.Add = addMap
					}
					if removeSlice, found, _ := unstructured.NestedStringSlice(modifierMap, "remove"); found {
						modifier.Remove = removeSlice
					}
					filter.RequestRewrite = modifier
				}
			case "ResponseHeaderModifier":
				if modifierMap, found, _ := unstructured.NestedMap(filterMap, "responseHeaderModifier"); found {
					modifier := &RouteResponseHeaderModifier{}
					if setMap, found, _ := unstructured.NestedStringMap(modifierMap, "set"); found {
						modifier.Set = setMap
					}
					if addMap, found, _ := unstructured.NestedStringMap(modifierMap, "add"); found {
						modifier.Add = addMap
					}
					if removeSlice, found, _ := unstructured.NestedStringSlice(modifierMap, "remove"); found {
						modifier.Remove = removeSlice
					}
					filter.ResponseRewrite = modifier
				}
			}

			filters = append(filters, filter)
		}
	}
	return filters
}

func parseBackendRefs(backendRefsRaw []any) []RouteBackendRef {
	var backendRefs []RouteBackendRef
	for _, backendRefRaw := range backendRefsRaw {
		if backendRefMap, ok := backendRefRaw.(map[string]any); ok {
			backendRef := RouteBackendRef{}
			if group, found, _ := unstructured.NestedString(backendRefMap, "group"); found {
				backendRef.Group = group
			}
			if kind, found, _ := unstructured.NestedString(backendRefMap, "kind"); found {
				backendRef.Kind = kind
			}
			if name, found, _ := unstructured.NestedString(backendRefMap, "name"); found {
				backendRef.Name = name
			}
			if namespace, found, _ := unstructured.NestedString(backendRefMap, "namespace"); found {
				backendRef.Namespace = namespace
			}
			if port, found, _ := unstructured.NestedInt64(backendRefMap, "port"); found {
				backendRef.Port = int32(port)
			}
			if weight, found, _ := unstructured.NestedInt64(backendRefMap, "weight"); found {
				backendRef.Weight = int32(weight)
			}
			backendRefs = append(backendRefs, backendRef)
		}
	}
	return backendRefs
}

func parseRouteStatus(statusMap map[string]any) RouteStatus {
	status := RouteStatus{}
	if parentsRaw, found, _ := unstructured.NestedSlice(statusMap, "parents"); found {
		var parents []RouteParentStatus
		for _, parentRaw := range parentsRaw {
			if parentMap, ok := parentRaw.(map[string]any); ok {
				parentStatus := RouteParentStatus{}

				// Parse ParentRef
				if parentRefMap, found, _ := unstructured.NestedMap(parentMap, "parentRef"); found {
					parentRef := RouteParentRef{}
					if group, found, _ := unstructured.NestedString(parentRefMap, "group"); found {
						parentRef.Group = group
					}
					if kind, found, _ := unstructured.NestedString(parentRefMap, "kind"); found {
						parentRef.Kind = kind
					}
					if name, found, _ := unstructured.NestedString(parentRefMap, "name"); found {
						parentRef.Name = name
					}
					if namespace, found, _ := unstructured.NestedString(parentRefMap, "namespace"); found {
						parentRef.Namespace = namespace
					}
					parentStatus.ParentRef = parentRef
				}

				// Parse ControllerName
				if controllerName, found, _ := unstructured.NestedString(parentMap, "controllerName"); found {
					parentStatus.ControllerName = controllerName
				}

				// Parse Conditions
				if conditionsRaw, found, _ := unstructured.NestedSlice(parentMap, "conditions"); found {
					var conditions []RouteCondition
					for _, conditionRaw := range conditionsRaw {
						if conditionMap, ok := conditionRaw.(map[string]any); ok {
							condition := RouteCondition{}
							if condType, found, _ := unstructured.NestedString(conditionMap, "type"); found {
								condition.Type = condType
							}
							if condStatus, found, _ := unstructured.NestedString(conditionMap, "status"); found {
								condition.Status = condStatus
							}
							if reason, found, _ := unstructured.NestedString(conditionMap, "reason"); found {
								condition.Reason = reason
							}
							conditions = append(conditions, condition)
						}
					}
					parentStatus.Conditions = conditions
				}

				parents = append(parents, parentStatus)
			}
		}
		status.Parents = parents
	}
	return status
}

type RoutesHandler struct {
	*BaseHandler
}

func NewRoutesHandler() *RoutesHandler {
	return &RoutesHandler{
		BaseHandler: &BaseHandler{
			name:          "routes",
			clusterScoped: false,
		},
	}
}

func (h *RoutesHandler) Collect(ctx context.Context, c *Collector, namespace string) ([]Resource, error) {
	routes, err := c.CollectRoutes(ctx, namespace)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	batch := make([]Resource, 0, len(routes))

	for _, route := range routes {
		batch = append(batch, Resource{
			Type:      "route",
			Namespace: namespace,
			Resource:  route,
			Timestamp: timestamp,
		})
	}

	return batch, nil
}
