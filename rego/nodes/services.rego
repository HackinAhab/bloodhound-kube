# Service Node Policies
# Creates BloodHound nodes for Kubernetes Services

package nodes.services

import rego.v1
import data.nodes.base
import data.nodes.helpers

# Main service node creation
nodes contains node if {
	some i
	resource := input.resources[i]
	resource.kind == "Service"
	metadata := base.extract_metadata(resource)
	
	selector_map := object.get(resource.spec, "selector", {})
	private := object.union(metadata.__private, {
		"selector_map": selector_map,
	})
	properties := object.union(metadata, {
		"serviceType": object.get(resource.spec, "type", "ClusterIP"),
		"clusterIP": object.get(resource.spec, "clusterIP", ""),
		"externalIPs": object.get(resource.spec, "externalIPs", []),
		"loadBalancerIP": object.get(resource.spec, "loadBalancerIP", ""),
		"selector": helpers.labels_map_to_list(selector_map),
		"__private": private,
		"ports": extract_ports(resource.spec),
		"sessionAffinity": object.get(resource.spec, "sessionAffinity", "None"),
		"externalTrafficPolicy": object.get(resource.spec, "externalTrafficPolicy", ""),
		"isHeadless": is_headless(resource.spec),
		"isExternal": is_external_service(resource.spec),
	})
	
	node := base.default_node("service", ["Service"], metadata.namespace, metadata.name, properties)
}

# Extract port information
extract_ports(spec) := ports if {
	spec.ports
	ports := [port |
		some i
		p := spec.ports[i]
		port := {
			"name": object.get(p, "name", ""),
			"protocol": object.get(p, "protocol", "TCP"),
			"port": p.port,
			"targetPort": p.targetPort,
			"nodePort": object.get(p, "nodePort", 0),
		}
	]
}

extract_ports(spec) := [] if {
	not spec.ports
}

# Check if service is headless
is_headless(spec) := true if {
	spec.clusterIP == "None"
} else := false

# Check if service is external-facing
is_external_service(spec) := true if {
	spec.type == "LoadBalancer"
} else := true if {
	spec.type == "NodePort"
} else := true if {
	count(spec.externalIPs) > 0
} else := false
