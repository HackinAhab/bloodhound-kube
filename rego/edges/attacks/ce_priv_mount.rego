# https://kubehound.io/reference/attacks/CE_PRIV_MOUNT/

package kubernetes.relationships.ce_privileged_mount
import data.kubernetes.helpers

ce_privileged_mount contains edge if {
    namespace := input.core.namespaces[ns]
    pod := namespace.pods[_]
    pod.nodeName != ""
    pod.id != ""

    privileged := pod.containers[_].privileged
    privileged == true

    node := input.core.cluster.nodes[_]
    node.name == pod.nodeName
    node.id != ""

    edge := helpers.create_edge_with_properties(pod, node, "CE_PRIV_MOUNT", {
        "Description": "Container in pod is privileged which may allow for mounting the host filesystem.",
        "Reference": "https://kubehound.io/reference/attacks/CE_PRIV_MOUNT/",
    })
}
