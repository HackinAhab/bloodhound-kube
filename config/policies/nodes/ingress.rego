# Ingress Node Policies
# Creates BloodHound nodes for Kubernetes Ingress resources

package nodes.ingress

import rego.v1
import data.nodes.base

# Main ingress node creation
nodes contains node if {
	some i
	resource := input.resources[i]
	resource.kind == "Ingress"
	metadata := base.extract_metadata(resource)
	
	properties := object.union(metadata, {
		"ingress_class": object.get(resource.spec, "ingressClassName", ""),
		"rules": extract_rules(resource.spec),
		"tls": extract_tls_config(resource.spec),
		"has_tls": has_tls_config(resource.spec),
		"default_backend": extract_default_backend(resource.spec),
	})
	
	node := base.default_node("ingress", ["Ingress"], metadata.namespace, metadata.name, properties)
}

# Extract ingress rules
extract_rules(spec) := rules if {
	spec.rules
	rules := [rule |
		some i
		r := spec.rules[i]
		rule := {
			"host": object.get(r, "host", ""),
			"paths": extract_paths(r),
		}
	]
}

extract_rules(spec) := [] if {
	not spec.rules
}

# Extract paths from rule
extract_paths(rule) := paths if {
	rule.http.paths
	paths := [path |
		some i
		p := rule.http.paths[i]
		path := {
			"path": object.get(p, "path", "/"),
			"path_type": object.get(p, "pathType", "Prefix"),
			"backend_service": object.get(p.backend.service, "name", ""),
			"backend_port": object.get(p.backend.service.port, "number", 0),
		}
	]
}

extract_paths(rule) := [] if {
	not rule.http.paths
}

# Extract TLS configuration
extract_tls_config(spec) := tls_configs if {
	spec.tls
	tls_configs := [tls |
		some i
		t := spec.tls[i]
		tls := {
			"hosts": object.get(t, "hosts", []),
			"secret_name": object.get(t, "secretName", ""),
		}
	]
}

extract_tls_config(spec) := [] if {
	not spec.tls
}

# Check if ingress has TLS
has_tls_config(spec) if {
	spec.tls
	count(spec.tls) > 0
}

# Extract default backend
extract_default_backend(spec) := backend if {
	spec.defaultBackend
	backend := {
		"service_name": object.get(spec.defaultBackend.service, "name", ""),
		"service_port": object.get(spec.defaultBackend.service.port, "number", 0),
	}
}

extract_default_backend(spec) := {} if {
	not spec.defaultBackend
}