package nodes.helpers

import future.keywords.in

# Generate node ID (matches existing format)
# For namespaced resources: Kind:namespace:name
# For cluster-scoped resources: Kind:name
generate_id(kind, namespace, name) := id {
    namespace != ""
    namespace != null
    id := sprintf("%s:%s:%s", [kind, namespace, name])
}

generate_id(kind, namespace, name) := id {
    namespace == ""
    id := sprintf("%s:%s", [kind, name])
}

generate_id(kind, namespace, name) := id {
    namespace == null
    id := sprintf("%s:%s", [kind, name])
}

# Detect sensitive keys in data
has_sensitive_keys(keys) {
    sensitive := ["password", "token", "key", "secret", "cert", "credential", "api_key", "apikey", "private"]
    some pattern in sensitive
    some key in keys
    contains(lower(key), pattern)
}

# Check if array contains any of the specified values
contains_any(array, values) {
    some i
    values[i]
    some j
    array[j]
    array[j] == values[i]
}

# Get all keys from a map/object
get_keys(obj) := keys {
    keys := object.keys(obj)
}

# Safe length function (returns 0 for null/missing)
safe_length(value) := count(value) {
    value != null
}

safe_length(value) := 0 {
    value == null
}

# Get labels safely
get_labels(resource) := labels {
    labels := object.get(resource, ["metadata", "labels"], {})
}

# Get annotations safely
get_annotations(resource) := annotations {
    annotations := object.get(resource, ["metadata", "annotations"], {})
}

# Get namespace safely
get_namespace(resource) := namespace {
    namespace := object.get(resource, ["metadata", "namespace"], "")
}

# Get name safely
get_name(resource) := name {
    name := object.get(resource, ["metadata", "name"], "")
}
