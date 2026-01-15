package nodes.cluster

import data.nodes.helpers

# Node → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "Node"
    
    addresses := [a | a := resource.status.addresses[_]]
    conditions := [c | c := resource.status.conditions[_]]
    taints := object.get(resource.spec, "taints", [])
    
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
            "unschedulable": object.get(resource.spec, "unschedulable", false),
            "pod_cidr": object.get(resource.spec, "podCIDR", ""),
            "provider_id": object.get(resource.spec, "providerID", ""),
        }
    }
}

# Namespace → Node
nodes[node] {
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
