# Service Node Policies
# Creates BloodHound nodes for Kubernetes Services

package nodes.services

import rego.v1
import data.nodes.base

# Main service node creation
nodes contains node if {
	some i
	resource := input.resources[i]
	resource.kind == "Service"
	metadata := base.extract_metadata(resource)
	
	properties := object.union(metadata, {
		"service_type": object.get(resource.spec, "type", "ClusterIP"),
		"cluster_ip": object.get(resource.spec, "clusterIP", ""),
		"external_ips": object.get(resource.spec, "externalIPs", []),
		"load_balancer_ip": object.get(resource.spec, "loadBalancerIP", ""),
		"selector": object.get(resource.spec, "selector", {}),
		"ports": extract_ports(resource.spec),
		"session_affinity": object.get(resource.spec, "sessionAffinity", "None"),
		"external_traffic_policy": object.get(resource.spec, "externalTrafficPolicy", ""),
		"is_headless": is_headless(resource.spec),
		"is_external": is_external_service(resource.spec),
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
			"target_port": p.targetPort,
			"node_port": object.get(p, "nodePort", 0),
		}
	]
}

extract_ports(spec) := [] if {
	not spec.ports
}

# Check if service is headless
is_headless(spec) if {
	spec.clusterIP == "None"
}

# Check if service is external-facing
is_external_service(spec) if {
	spec.type == "LoadBalancer"
}

is_external_service(spec) if {
	spec.type == "NodePort"
}

is_external_service(spec) if {
	count(spec.externalIPs) > 0
}