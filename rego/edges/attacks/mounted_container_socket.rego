# https://kubehound.io/reference/attacks/EXPLOIT_CONTAINERD_SOCK/

package kubernetes.relationships.mounted_container_socket
import data.kubernetes.helpers


mounted_container_socket contains edge if {
    namespace := input.namespaces[ns]
    pod := namespace.pod[_]
    pod.properties.nodeName != ""
    pod.id != ""

    pod.properties.__private.volumes[_].hostPath != ""
    host_path := pod.properties.__private.volumes[_].hostPath
    # Note: This is really simplified, its only checking for socket files, which might not be the only way container runtimes are exposed, so this will produce false positives.
    # TODO: Enhance this to check for known container runtime socket paths, and their parent directories.
    endswith(host_path, ".sock")

    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName
    node.id != ""

    edge := helpers.create_edge_with_properties(pod, node, "MOUNTED_CONTAINER_SOCKET", {
        "Description": sprintf("Container in pod has a hostPath volume mount to a path that potentially contains a container socket: %s. ", [host_path]),
        "Reference": "https://kubehound.io/reference/attacks/EXPLOIT_CONTAINERD_SOCK/",
    })
}