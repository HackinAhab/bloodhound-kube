# https://kubehound.io/reference/attacks/CE_NSENTER/

package kubernetes.relationships.ce_nsenter
import data.kubernetes.helpers

ce_ns_enter contains edge if {
    namespace := input.namespaces[ns]
    pod := namespace.pod[_]
    pod.properties.nodeName != ""
    pod.id != ""

    privileged := pod.properties.__private.containers[_].privileged
    privileged == true
    hostPID := pod.properties.hostPID
    hostPID == true

    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName
    node.id != ""

    edge := helpers.create_edge_with_properties(pod, node, "CE_NSENTER", {
        "Description": "Container in pod is privileged and has hostPID enabled which may allow for escaping the container and executing commands on the host using nsenter.",
        "Reference": "https://kubehound.io/reference/attacks/CE_NSENTER/",
    })
}