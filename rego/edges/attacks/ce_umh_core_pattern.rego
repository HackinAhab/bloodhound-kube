# https://kubehound.io/reference/attacks/CE_UMH_CORE_PATTERN/

package kubernetes.relationships.ce_umh_core_pattern
import data.kubernetes.helpers

ce_umh_core_pattern contains edge if {
    namespace := input.core.namespaces[ns]
    pod := namespace.pods[_]
    pod.nodeName != ""
    pod.id != ""

    pod.volumes[_].hostPath != ""
    host_path := pod.volumes[_].hostPath
    host_path in ["/proc","/proc/sys","/proc/sys/kernel"]

    pod.containers[_].volumeMounts[_].readOnly != true
    mount_path := pod.containers[_].volumeMounts[_].mountPath

    node := input.core.cluster.nodes[_]
    node.name == pod.nodeName
    node.id != ""

    edge := helpers.create_edge_with_properties(pod, node, "CE_UMH_CORE_PATTERN", {
        "Description": sprintf("Container in pod has a hostPath volume mount to a critical procfs path which may allow for container escape via usermode helper pattern. Note: this check does not verify if the container is running as the root user, which will likely be required to write to the /proc/sys/kernel/core_pattern file. Mount path: %s", [mount_path]),
        "Reference": "https://kubehound.io/reference/attacks/CE_UMH_CORE_PATTERN/",
    })
}
