package nodes.networking

import data.nodes.helpers

# Service → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "Service"
    
    selector := object.get(resource.spec, "selector", {})
    ports := [p | p := resource.spec.ports[_]]
    
    node := {
        "id": helpers.generate_id("Service", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["Service"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "service",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "selector": selector,
            "ports": ports,
            "service_type": object.get(resource.spec, "type", "ClusterIP"),
            "cluster_ip": object.get(resource.spec, "clusterIP", ""),
            "external_ips": object.get(resource.spec, "externalIPs", []),
        }
    }
}

# Ingress → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "Ingress"
    
    rules := [r | r := resource.spec.rules[_]]
    tls := object.get(resource.spec, "tls", [])
    
    node := {
        "id": helpers.generate_id("Ingress", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["Ingress"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "ingress",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "rules": rules,
            "tls": tls,
            "has_tls": count(tls) > 0,
            "ingress_class": object.get(resource.spec, "ingressClassName", object.get(resource.metadata.annotations, "kubernetes.io/ingress.class", "")),
        }
    }
}

# NetworkPolicy → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "NetworkPolicy"
    
    pod_selector := object.get(resource.spec, "podSelector", {})
    ingress_rules := object.get(resource.spec, "ingress", [])
    egress_rules := object.get(resource.spec, "egress", [])
    policy_types := object.get(resource.spec, "policyTypes", [])
    
    node := {
        "id": helpers.generate_id("NetworkPolicy", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["NetworkPolicy"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "networkpolicy",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "pod_selector": pod_selector,
            "ingress_rules": ingress_rules,
            "egress_rules": egress_rules,
            "policy_types": policy_types,
            "has_ingress_rules": count(ingress_rules) > 0,
            "has_egress_rules": count(egress_rules) > 0,
        }
    }
}
