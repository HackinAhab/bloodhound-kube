package nodes.workloads

import data.nodes.helpers

# Pod → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "Pod"
    
    volumes := [v | v := resource.spec.volumes[_]]
    containers := [c | c := resource.spec.containers[_]]
    images := [c.image | c := resource.spec.containers[_]]
    
    node := {
        "id": helpers.generate_id("Pod", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["Pod"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "pod",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "service_account": object.get(resource.spec, "serviceAccountName", ""),
            "volumes": volumes,
            "containers": containers,
            "images": images,
            "host_network": object.get(resource.spec, "hostNetwork", false),
            "host_pid": object.get(resource.spec, "hostPID", false),
            "host_ipc": object.get(resource.spec, "hostIPC", false),
            "node_name": object.get(resource.spec, "nodeName", ""),
        }
    }
}

# Deployment → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "Deployment"
    
    selector := object.get(resource.spec, ["selector", "matchLabels"], {})
    
    node := {
        "id": helpers.generate_id("Deployment", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["Deployment"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "deployment",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "replicas": object.get(resource.spec, "replicas", 1),
            "selector": selector,
            "strategy": object.get(resource.spec, ["strategy", "type"], "RollingUpdate"),
        }
    }
}

# StatefulSet → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "StatefulSet"
    
    selector := object.get(resource.spec, ["selector", "matchLabels"], {})
    
    node := {
        "id": helpers.generate_id("StatefulSet", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["StatefulSet"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "statefulset",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "replicas": object.get(resource.spec, "replicas", 1),
            "selector": selector,
            "service_name": object.get(resource.spec, "serviceName", ""),
        }
    }
}

# DaemonSet → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "DaemonSet"
    
    selector := object.get(resource.spec, ["selector", "matchLabels"], {})
    
    node := {
        "id": helpers.generate_id("DaemonSet", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["DaemonSet"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "daemonset",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "selector": selector,
            "update_strategy": object.get(resource.spec, ["updateStrategy", "type"], "RollingUpdate"),
        }
    }
}
