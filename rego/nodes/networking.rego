package nodes.networking

import rego.v1

import data.nodes.helpers

# NetworkPolicy → Node
nodes contains node if {
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
	"labels": helpers.labels_to_list(resource),
	"annotations": helpers.annotations_to_list(resource),
	"labels_map": helpers.get_labels(resource),
	"annotations_map": helpers.get_annotations(resource),
            "pod_selector": pod_selector,
            "ingress_rules": ingress_rules,
            "egress_rules": egress_rules,
            "policy_types": policy_types,
            "has_ingress_rules": count(ingress_rules) > 0,
            "has_egress_rules": count(egress_rules) > 0,
        }
    }
}
