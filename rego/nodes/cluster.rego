package nodes.cluster

import rego.v1

import data.nodes.helpers

# Node → Node
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "Node"
    
    status := object.get(resource, "status", {})
    spec := object.get(resource, "spec", {})

    addresses := object.get(status, "addresses", [])
    conditions := object.get(status, "conditions", [])
    taints := object.get(spec, "taints", [])
    
    node := {
        "id": helpers.generate_id("Node", "", resource.metadata.name),
        "kinds": ["Node"],
        "properties": {
            "name": helpers.get_name(resource),
            "resource_type": "node",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "addresses": addresses,
            "conditions": conditions,
            "taints": taints,
            "unschedulable": object.get(spec, "unschedulable", false),
            "pod_cidr": object.get(spec, "podCIDR", ""),
            "provider_id": object.get(spec, "providerID", ""),
        }
    }
}

# NodeList → Node
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "NodeList"

    item := resource.items[_]

    status := object.get(item, "status", {})
    spec := object.get(item, "spec", {})

    addresses := object.get(status, "addresses", [])
    conditions := object.get(status, "conditions", [])
    taints := object.get(spec, "taints", [])

    node := {
        "id": helpers.generate_id("Node", "", item.metadata.name),
        "kinds": ["Node"],
        "properties": {
            "name": helpers.get_name(item),
            "resource_type": "node",
            "labels": helpers.get_labels(item),
            "annotations": helpers.get_annotations(item),
            "addresses": addresses,
            "conditions": conditions,
            "taints": taints,
            "unschedulable": object.get(spec, "unschedulable", false),
            "pod_cidr": object.get(spec, "podCIDR", ""),
            "provider_id": object.get(spec, "providerID", ""),
        }
    }
}

# Namespace → Node
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "Namespace"
    
    node := {
        "id": helpers.generate_id("Namespace", "", resource.metadata.name),
        "kinds": ["Namespace"],
        "properties": {
            "name": helpers.get_name(resource),
            "resource_type": "namespace",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "status": object.get(resource.status, "phase", "Active"),
        }
    }
}
