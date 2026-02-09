# Base Node Policy
# Defines core node creation patterns and utilities for all Kubernetes resources

package nodes.base

import rego.v1

import data.nodes.helpers

# Default node structure
default_node(resource_type, kinds, namespace, name, properties) := node if {
	node := {
		"id": helpers.generate_id(kinds[0], namespace, name),
		"kinds": kinds,
		"properties": object.union(properties, {
			"name": name,
			"namespace": namespace,
			"resource_type": resource_type,
		}),
	}
}

# Generate consistent node IDs
generate_node_id(resource_type, namespace, name) := id if {
	namespace != ""
	id := sprintf("%s/%s/%s", [resource_type, namespace, name])
}

generate_node_id(resource_type, namespace, name) := id if {
	namespace == ""
	id := sprintf("%s/cluster/%s", [resource_type, name])
}

# Extract common metadata
extract_metadata(resource) := metadata if {
	metadata := {
		"name": resource.metadata.name,
		"namespace": object.get(resource.metadata, "namespace", ""),
		"uid": object.get(resource.metadata, "uid", ""),
		"labels": helpers.labels_to_list(resource),
		"annotations": helpers.annotations_to_list(resource),
		"labels_map": object.get(resource.metadata, "labels", {}),
		"annotations_map": object.get(resource.metadata, "annotations", {}),
		"created_at": object.get(resource.metadata, "creationTimestamp", ""),
	}
}

# Check if resource has label
has_label(resource, key) if {
	resource.metadata.labels[key]
}

# Check if resource has annotation
has_annotation(resource, key) if {
	resource.metadata.annotations[key]
}

# Extract owner references
extract_owners(resource) := owners if {
	owners := [owner |
		some i
		owner_ref := resource.metadata.ownerReferences[i]
		owner := {
			"kind": owner_ref.kind,
			"name": owner_ref.name,
			"uid": owner_ref.uid,
		}
	]
}

extract_owners(resource) := [] if {
	not resource.metadata.ownerReferences
}

# Check if resource is namespaced
is_namespaced(resource) if {
	resource.metadata.namespace
	resource.metadata.namespace != ""
}

# Common security checks
is_privileged(container) if {
	container.securityContext.privileged == true
}

runs_as_root(container) if {
	not container.securityContext.runAsNonRoot
}

runs_as_root(container) if {
	container.securityContext.runAsUser == 0
}

has_host_network(pod_spec) if {
	pod_spec.hostNetwork == true
}

has_host_pid(pod_spec) if {
	pod_spec.hostPID == true
}

has_host_ipc(pod_spec) if {
	pod_spec.hostIPC == true
}

# Extract sensitive capabilities
dangerous_capabilities := [
	"SYS_ADMIN",
	"NET_ADMIN",
	"SYS_MODULE",
	"SYS_RAWIO",
	"SYS_PTRACE",
	"SYS_BOOT",
	"MAC_ADMIN",
	"MAC_OVERRIDE",
	"PERFMON",
	"BPF",
]

has_dangerous_capabilities(container) if {
	cap := container.securityContext.capabilities.add[_]
	cap in dangerous_capabilities
}

# Volume type checks
is_host_path_volume(volume) if {
	volume.hostPath
}

is_secret_volume(volume) if {
	volume.secret
}

is_configmap_volume(volume) if {
	volume.configMap
}

is_pvc_volume(volume) if {
	volume.persistentVolumeClaim
}
