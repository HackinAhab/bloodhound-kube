package nodes.statefulsets

import rego.v1

import data.nodes.helpers

# StatefulSet → Node
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "StatefulSet"

    selector := object.get(resource.spec, ["selector", "matchLabels"], {})

    node := {
        "id": helpers.generate_id("StatefulSet", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["StatefulSet"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "statefulset",
		"labels": helpers.labels_to_list(resource),
		"annotations": helpers.annotations_to_list(resource),
		"labels_map": helpers.get_labels(resource),
		"annotations_map": helpers.get_annotations(resource),
            "selector": selector,
            "service_name": object.get(resource.spec, "serviceName", ""),
        }
    }
}
