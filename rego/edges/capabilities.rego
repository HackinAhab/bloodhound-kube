package kubernetes.relationships.capabilities

import rego.v1
import data.kubernetes.helpers

# Pod with dangerous capabilities scheduled on Node
pod_dangerous_caps_on_node contains edge if {
    namespace := input.namespaces[ns]
    pod := namespace.pod[_]
    pod.properties.nodeName != ""
    pod.id != ""

    cap_add := pod.properties.__private.capabilitiesAdd[_]
    norm := helpers.normalize_capability(cap_add)
    helpers.capability_descriptions[norm]

    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName
    node.id != ""
    description := sprintf("Container in pod %s has dangerous capabilities that could allow for privilege escalation or container escape.", [pod.properties.name])
    edge := helpers.create_edge_with_properties(pod, node, "DangerousCaps", {
        "Description": description,
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
    description := helpers.capability_descriptions[norm]

    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName
    node.id != ""

    edge := helpers.create_edge_with_properties(pod, node, norm, {
        "Description": description,
    })
}
