package kubernetes.helpers

# Check if map1 is subset of map2 (for label selectors)
# All keys in selector must exist in labels with matching values
is_subset(selector, labels) {
	selector_keys := object.keys(selector)
	matching_keys := {k | k := selector_keys[_]; selector[k] == labels[k]}
	count(matching_keys) == count(selector_keys)
}

# Check if value exists in array
contains_value(array, value) {
	array[_] == value
}

# Check if array contains any of the specified values
contains_any(array, values) {
	some i
	values[i]
	contains_value(array, values[i])
}

# Create standard edge structure
create_edge(source, target, kind, priority) := edge {
	edge := {
		"start": {
			"match_by": "id",
			"value": source.id,
		},
		"end": {
			"match_by": "id",
			"value": target.id,
		},
		"kind": kind,
		"properties": {"priority": priority},
	}
}

# Create edge with additional via information (for via-based relationships)
create_edge_via(source, target, via, kind, priority) := edge {
	edge := {
		"start": {
			"match_by": "id",
			"value": source.id,
		},
		"end": {
			"match_by": "id",
			"value": target.id,
		},
		"kind": kind,
		"properties": {
			"priority": priority,
			"via": via.id,
			"via_kind": via.kinds[0],
		},
	}
}

# Get property value safely with default
get_property(obj, key, default_value) := value {
	value := obj[key]
} else := default_value

# Check if object has property
has_property(obj, key) {
	_ := obj[key]
}
