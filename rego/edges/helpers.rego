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

# Create edge with properties
create_edge_with_properties(source, target, edge_type, props) := edge if {
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
		"properties": props,
	}
}

# Capability descriptions (add entries as needed)
capability_descriptions := {
	"CAP_SYS_ADMIN": {
		"Description": "Container in pod has CAP_SYS_ADMIN capability which is a powerful capability that can allow for a wide range of actions, including privilege escalation and container escape.",
		"Reference": "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_admin",
	},
	"CAP_NET_ADMIN": {
		"Description": "Container in pod has CAP_NET_ADMIN capability which allows for network administration tasks and can be used for malicious purposes such as intercepting network traffic or modifying network configurations.",
		"Reference": "",
	},
	"CAP_SYS_MODULE": {
		"Description": "Container in pod has CAP_SYS_MODULE capability which allows for loading and unloading kernel modules, and can be used for malicious purposes such as installing rootkits or other kernel-level malware.",
		"Reference": "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_module",
	},
	"CAP_SYS_PTRACE": {
		"Description": "Container in pod has CAP_SYS_PTRACE capability which allows for tracing and debugging of processes, and can be used for malicious purposes such as stealing sensitive information from other processes or performing code injection attacks.",
		"Reference": "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_ptrace",
	},
	"CAP_SYS_RAWIO": {
		"Description": "Container in pod has CAP_SYS_RAWIO capability which allows for raw I/O operations, and can be used for malicious purposes such as bypassing security controls or accessing sensitive data on the host.",
		"Reference": "https://book.hacktricks.wiki/en/linux-hardening/privilege-escalation/linux-capabilities.html#cap_sys_rawio",
	},
}

normalize_capability(cap) := cap if {
	startswith(cap, "CAP_")
}

normalize_capability(cap) := norm if {
	not startswith(cap, "CAP_")
	norm := sprintf("CAP_%s", [cap])
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
