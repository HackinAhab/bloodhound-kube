# Relationship Helpers
# Common utilities for creating edges between nodes

package kubernetes.helpers

import rego.v1

# Create edge
create_edge(source, target, edge_type) := edge if {
	edge := {
		"start": {
			"match_by": "id",
			"value": source.id,
			"kind": source.kinds[0],
		},
		"end": {
			"match_by": "id",
			"value": target.id,
			"kind": target.kinds[0],
		},
		"kind": edge_type,
		"properties": {},
	}
}

# Create edge with via node
create_edge_via(source, target, via, edge_type) := edge if {
	edge := {
		"start": {
			"match_by": "id",
			"value": source.id,
			"kind": source.kinds[0],
		},
		"end": {
			"match_by": "id",
			"value": target.id,
			"kind": target.kinds[0],
		},
		"kind": edge_type,
		"properties": {
			"via_id": via.id,
			"via_kind": via.kinds[0],
			"via_name": via.properties.name,
		},
	}
}

# Check if labels match selector (subset matching)
labels_match_selector(labels, selector) if {
	# Every key in selector must exist in labels with the same value
	every key, value in selector {
		labels[key] == value
	}
}

# Check if map is a subset of another map
is_subset(subset, superset) if {
	every key, value in subset {
		superset[key] == value
	}
}

# Extract service accounts from subjects
extract_service_accounts(subjects, namespace) := sas if {
	sas := [sa |
		some i
		subject := subjects[i]
		subject.kind == "ServiceAccount"
		sa := {
			"name": subject.name,
			"namespace": object.get(subject, "namespace", namespace),
		}
	]
}

# Check if volume references secret
volume_references_secret(volume, secret_name) if {
	volume.secret.secretName == secret_name
}

# Check if volume references configmap
volume_references_configmap(volume, configmap_name) if {
	volume.configMap.name == configmap_name
}

# Get namespace safely
get_namespace(resource) := namespace if {
	namespace := object.get(resource.properties, "namespace", "")
}

# Check if resources are in same namespace
same_namespace(resource1, resource2) if {
	get_namespace(resource1) == get_namespace(resource2)
	get_namespace(resource1) != ""
}
