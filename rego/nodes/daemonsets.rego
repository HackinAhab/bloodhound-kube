package nodes.daemonsets

import rego.v1

import data.nodes.helpers

# DaemonSet → Node
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "DaemonSet"

    selector := object.get(resource.spec, ["selector", "matchLabels"], {})

    node := {
        "id": helpers.generate_id("DaemonSet", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["DaemonSet"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "daemonset",
		"labels": helpers.labels_to_list(resource),
		"annotations": helpers.annotations_to_list(resource),
		"labels_map": helpers.get_labels(resource),
		"annotations_map": helpers.get_annotations(resource),
            "selector": selector,
            "update_strategy": object.get(resource.spec, ["updateStrategy", "type"], "RollingUpdate"),
        }
    }
}
