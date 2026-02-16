# Based losely on https://github.com/aquasecurity/trivy-checks/tree/main/checks/kubernetes/access_to_host_ports.rego
package kubernetes.relationships.host_ports

import rego.v1

import data.kubernetes.helpers

pod_with_host_port contains edge if {
    namespace := input.namespaces[ns]
    pod := namespace.pod[_]

    host_ports := pod.properties.__private.containers[_].hostPorts[_]
    host_port := host_ports.hostPort
    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName

    edge := helpers.create_edge_with_properties(node, pod, "HostPort", {
        "HostPort": host_port,
    })
}

external_to_node_with_host_port contains edge if {
    namespace := input.namespaces[ns]
    pod := namespace.pod[_]

    host_ports := pod.properties.__private.containers[_].hostPorts[_]
    host_port := host_ports.hostPort
    node := input.cluster_scoped.node[_]
    node.properties.name == pod.properties.nodeName
    external := input.cluster_scoped.external[_]

    edge := helpers.create_edge_with_properties(external, node, "ExternalHostPort", {
        "Description": sprintf("External access to node %s via host port %d", [node.properties.name, host_port]),
    })
}
