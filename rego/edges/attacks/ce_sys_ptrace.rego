# Reference: https://kubehound.io/reference/attacks/CE_SYS_PTRACE/

package kubernetes.relationships.ce_sys_ptrace
import data.kubernetes.helpers

ce_sys_ptrace contains edge if {
    namespace := input.namespaces[ns]
    pod := namespace.pod[_]
    pod.properties.nodeName != ""
    pod.id != ""

    host_pid := pod.properties.hostPid == true
    has_caps := helpers.has_capability(pod, "CAP_SYS_PTRACE")
    has_admin := helpers.has_capability(pod, "CAP_SYS_ADMIN")
    privileged := pod.properties.__private.containers[_].privileged == true

    is_vulnerable(host_pid, has_caps, has_admin, privileged)

    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName
    node.id != ""

    edge := helpers.create_edge_with_properties(pod, node, "CE_SYS_PTRACE", {
        "Description": "Container in pod has CAP_SYS_PTRACE, and CAP_SYS_ADMIN capabilities, and has hostPID: True, or is privileged which allows for tracing and debugging of processes, and can be used to escape the container by attaching to processes running on the host.",
        "Reference": "https://kubehound.io/reference/attacks/CE_SYS_PTRACE/",
    })
}

is_vulnerable(host_pid, has_caps, has_admin, privileged) if {
	privileged
}

is_vulnerable(host_pid, has_caps, has_admin, privileged) if {
	host_pid
	has_caps
	has_admin
}
