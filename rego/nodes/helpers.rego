package nodes.helpers

import rego.v1

import future.keywords.in

# Generate node ID (matches existing format)
# For namespaced resources: Kind:namespace:name
# For cluster-scoped resources: Kind:name
generate_id(kind, namespace, name) := id if {
    namespace != ""
    namespace != null
    id := sprintf("%s:%s:%s", [kind, namespace, name])
}

generate_id(kind, namespace, name) := id if {
    namespace == ""
    id := sprintf("%s:%s", [kind, name])
}

generate_id(kind, namespace, name) := id if {
    namespace == null
    id := sprintf("%s:%s", [kind, name])
}

# Detect sensitive keys in data
has_sensitive_keys(keys) if {
    sensitive := ["password", "token", "key", "secret", "cert", "credential", "api_key", "apikey", "private"]
    some pattern in sensitive
    some key in keys
    contains(lower(key), pattern)
}

# Check if array contains any of the specified values
contains_any(array, values) if {
    some i
    values[i]
    some j
    array[j]
    array[j] == values[i]
}

# Get all keys from a map/object
get_keys(obj) := keys if {
    keys := object.keys(obj)
}

# Safe length function (returns 0 for null/missing)
safe_length(value) := count(value) if {
    value != null
}

safe_length(value) := 0 if {
    value == null
}

# Get labels safely
get_labels(resource) := labels if {
    labels := object.get(resource, ["metadata", "labels"], {})
}

# Get annotations safely
get_annotations(resource) := annotations if {
    annotations := object.get(resource, ["metadata", "annotations"], {})
}

# Convert label map to sorted list of "key=value"
labels_map_to_list(labels) := items if {
	keys := sort(object.keys(labels))
	items := [sprintf("%s=%v", [k, labels[k]]) | k := keys[_]]
}

# Convert annotation map to sorted list of "key=value"
annotations_map_to_list(annotations) := items if {
	keys := sort(object.keys(annotations))
	items := [sprintf("%s=%v", [k, annotations[k]]) | k := keys[_]]
}

# Convert labels to sorted list of "key=value"
labels_to_list(resource) := items if {
	labels := get_labels(resource)
	keys := sort(object.keys(labels))
	items := [sprintf("%s=%v", [k, labels[k]]) | k := keys[_]]
}

# Convert annotations to sorted list of "key=value"
annotations_to_list(resource) := items if {
	annotations := get_annotations(resource)
	keys := sort(object.keys(annotations))
	items := [sprintf("%s=%v", [k, annotations[k]]) | k := keys[_]]
}

# Get namespace safely
get_namespace(resource) := namespace if {
    namespace := object.get(resource, ["metadata", "namespace"], "")
}

# Get name safely
get_name(resource) := name if {
    name := object.get(resource, ["metadata", "name"], "")
}

# Known resource kinds with node policies
known_kinds := {
    "clusterrole",
    "clusterrolebinding",
    "clustersecretstore",
    "configmap",
    "daemonset",
    "deployment",
    "externalsecret",
    "httproute",
    "ingress",
    "namespace",
    "networkpolicy",
    "node",
    "nodelist",
    "persistentvolume",
    "persistentvolumeclaim",
    "pod",
    "role",
    "rolebinding",
    "secret",
    "secretstore",
    "securitycontextconstraints",
    "service",
    "serviceaccount",
    "statefulset",
}

is_known_kind(kind) if {
    kind != null
    lower(kind) in known_kinds
}
