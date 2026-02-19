package kubernetes.relationships.capabilities

import rego.v1
import data.kubernetes.helpers


# Pod with specific capability scheduled on Node
pod_capability_on_node contains edge if {
    namespace := input.core.namespaces[ns]
    pod := namespace.pods[_]
    pod.nodeName != ""
    pod.id != ""

    cap_add := pod.capabilitiesAdd[_]
    norm := helpers.normalize_capability(cap_add)
    entry := helpers.capability_descriptions[norm]

    node := input.core.cluster.nodes[_]
    node.name == pod.nodeName
    node.id != ""

    edge := helpers.create_edge_with_properties(pod, node, norm, {
        "Description": entry.Description,
        "Reference": entry.Reference,
    })
}
