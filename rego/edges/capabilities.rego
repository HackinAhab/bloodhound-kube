package kubernetes.relationships.capabilities

import rego.v1
import data.kubernetes.helpers


# Pod with specific capability scheduled on Node
pod_capability_on_node contains edge if {
    namespace := input.namespaces[ns]
    pod := namespace.pod[_]
    pod.properties.nodeName != ""
    pod.id != ""

    cap_add := pod.properties.__private.capabilitiesAdd[_]
    norm := helpers.normalize_capability(cap_add)
    entry := helpers.capability_descriptions[norm]

    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName
    node.id != ""

    edge := helpers.create_edge_with_properties(pod, node, norm, {
        "Description": entry.Description,
        "Reference": entry.Reference,
    })
}
