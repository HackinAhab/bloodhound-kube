# https://kubehound.io/reference/attacks/CE_PRIV_MOUNT/

package kubernetes.relationships.ce_privileged_mount
import data.kubernetes.helpers

ce_privileged_mount contains edge if {
    namespace := input.namespaces[ns]
    pod := namespace.pod[_]
    pod.properties.nodeName != ""
    pod.id != ""

    privileged := pod.properties.__private.containers[_].privileged
    privileged != false

    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName
    node.id != ""

    edge := helpers.create_edge_with_properties(pod, node, "CE_PRIV_MOUNT", {
        "Description": "Container in pod is privileged which may allow for mounting the host filesystem.",
        "Reference": "https://kubehound.io/reference/attacks/CE_PRIV_MOUNT/",
    })
}