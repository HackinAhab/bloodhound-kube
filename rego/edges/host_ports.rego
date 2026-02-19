# Based losely on https://github.com/aquasecurity/trivy-checks/tree/main/checks/kubernetes/access_to_host_ports.rego
package kubernetes.relationships.host_ports

import rego.v1

import data.kubernetes.helpers

pod_with_host_port contains edge if {
    namespace := input.core.namespaces[ns]
    pod := namespace.pods[_]

    host_ports := pod.containers[_].hostPorts[_]
    host_port := host_ports.hostPort
    node := input.core.cluster.nodes[_]
    node.name == pod.nodeName

    edge := helpers.create_edge_with_properties(node, pod, "HostPort", {
        "HostPort": host_port,
    })
}

external_to_node_with_host_port contains edge if {
    namespace := input.core.namespaces[ns]
    pod := namespace.pods[_]

    host_ports := pod.containers[_].hostPorts[_]
    host_port := host_ports.hostPort
    node := input.core.cluster.nodes[_]
    node.name == pod.nodeName
    external := input.cluster_scoped.external[_]

    edge := helpers.create_edge_with_properties(external, node, "ExternalHostPort", {
        "Description": sprintf("External access to node %s via host port %d", [node.name, host_port]),
    })
}
