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
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "replicas": object.get(resource.spec, "replicas", 1),
            "selector": selector,
            "service_name": object.get(resource.spec, "serviceName", ""),
        }
    }
}
