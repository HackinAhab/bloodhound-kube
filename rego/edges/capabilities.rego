package kubernetes.relationships.capabilities

import rego.v1
import data.kubernetes.helpers

# Pod with dangerous capabilities scheduled on Node
# TBD if this in addition to the specific capability edges is still useful.
# It might be useful to keeep the general "DangerousCaps" edge for quick identification of risky pods, while the specific capability edges provide more detailed information about which capabilities are involved.
pod_dangerous_caps_on_node contains edge if {
    namespace := input.namespaces[ns]
    pod := namespace.pod[_]
    pod.properties.nodeName != ""
    pod.id != ""

    cap_add := pod.properties.__private.capabilitiesAdd[_]
    norm := helpers.normalize_capability(cap_add)
    helpers.capability_descriptions[norm]
    entry := helpers.capability_descriptions[norm]

    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName
    node.id != ""
    description := sprintf("Container in pod %s has dangerous capabilities that could allow for privilege escalation or container escape.", [pod.properties.name])
    edge := helpers.create_edge_with_properties(pod, node, "DangerousCaps", {
        "Description": description,
        "Reference": entry.Reference,
    })
}

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
