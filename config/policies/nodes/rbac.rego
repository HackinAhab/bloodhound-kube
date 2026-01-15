package nodes.rbac

import data.nodes.helpers

# ServiceAccount → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "ServiceAccount"
    
    secrets := [s.name | s := object.get(resource, "secrets", [])[_]]
    image_pull_secrets := [s.name | s := object.get(resource, "imagePullSecrets", [])[_]]
    
    node := {
        "id": helpers.generate_id("ServiceAccount", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["ServiceAccount"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "serviceaccount",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "secrets": secrets,
            "image_pull_secrets": image_pull_secrets,
            "automount_service_account_token": object.get(resource, "automountServiceAccountToken", true),
        }
    }
}

# Role → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "Role"
    
    rules := object.get(resource, "rules", [])
    
    node := {
        "id": helpers.generate_id("Role", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["Role"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "role",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "rules": rules,
            "rules_count": count(rules),
        }
    }
}

# ClusterRole → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "ClusterRole"
    
    rules := object.get(resource, "rules", [])
    
    node := {
        "id": helpers.generate_id("ClusterRole", "", resource.metadata.name),
        "kinds": ["ClusterRole"],
        "properties": {
            "name": helpers.get_name(resource),
            "resource_type": "clusterrole",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "rules": rules,
            "rules_count": count(rules),
        }
    }
}

# RoleBinding → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "RoleBinding"
    
    subjects := object.get(resource, "subjects", [])
    role_ref := object.get(resource, "roleRef", {})
    
    node := {
        "id": helpers.generate_id("RoleBinding", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["RoleBinding"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "rolebinding",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "subjects": subjects,
            "role_ref": role_ref,
            "role_name": object.get(role_ref, "name", ""),
            "role_kind": object.get(role_ref, "kind", ""),
        }
    }
}

# ClusterRoleBinding → Node
nodes[node] {
    resource := input.resources[_]
    resource.kind == "ClusterRoleBinding"
    
    subjects := object.get(resource, "subjects", [])
    role_ref := object.get(resource, "roleRef", {})
    
    node := {
        "id": helpers.generate_id("ClusterRoleBinding", "", resource.metadata.name),
        "kinds": ["ClusterRoleBinding"],
        "properties": {
            "name": helpers.get_name(resource),
            "resource_type": "clusterrolebinding",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "subjects": subjects,
            "role_ref": role_ref,
            "role_name": object.get(role_ref, "name", ""),
            "role_kind": object.get(role_ref, "kind", ""),
        }
    }
}
