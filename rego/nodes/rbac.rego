package nodes.rbac

import rego.v1

import data.nodes.helpers

# Build permissions list: "<apiGroup>/<resource>[/<resourceName>]: verb,verb"
perms(rules) := perm_list if {
	resource_map := perms_map(rules)
	non_resource_map := {url: verbs |
		rule := rules[_]
		url := rule.nonResourceURLs[_]
		verbs := [v |
			rule2 := rules[_]
			url2 := rule2.nonResourceURLs[_]
			url2 == url
			v := rule2.verbs[_]
		]
	}
	resource_keys := [k | k := object.keys(resource_map)[_]]
	non_resource_keys := [k | k := object.keys(non_resource_map)[_]]
	keys := sort(array.concat(resource_keys, non_resource_keys))
	perm_list := [sprintf("%s: %s", [key, concat(", ", sort(object.get(resource_map, key, object.get(non_resource_map, key, []))))]) |
		key := keys[_]
	]
}

perms_map(rules) := rules_map if {
	rules_map := {key: verbs |
		rule := rules[_]
		resource := rule.resources[_]
		group := object.get(rule, "apiGroups", [""])[_]
		name := object.get(rule, "resourceNames", [""])[_]
		key := resource_key(group, resource, name)
		verbs := [v |
			rule2 := rules[_]
			resource2 := rule2.resources[_]
			group2 := object.get(rule2, "apiGroups", [""])[_]
			name2 := object.get(rule2, "resourceNames", [""])[_]
			resource_key(group2, resource2, name2) == key
			v := rule2.verbs[_]
		]
	}
}

resource_key(group, resource, name) := key if {
	group != ""
	name == ""
	key := sprintf("%s/%s", [group, resource])
}

resource_key(group, resource, name) := key if {
	group == ""
	name == ""
	key := resource
}

resource_key(group, resource, name) := key if {
	name != ""
	base := resource_base_key(group, resource)
	key := sprintf("%s/%s", [base, name])
}

resource_base_key(group, resource) := key if {
	group != ""
	key := sprintf("%s/%s", [group, resource])
}

resource_base_key(group, resource) := key if {
	group == ""
	key := resource
}


# ServiceAccount → Node
nodes contains node if {
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
		"labels": helpers.labels_to_list(resource),
		"annotations": helpers.annotations_to_list(resource),
		"labels_map": helpers.get_labels(resource),
		"annotations_map": helpers.get_annotations(resource),
            "secrets": secrets,
            "image_pull_secrets": image_pull_secrets,
            "automount_service_account_token": object.get(resource, "automountServiceAccountToken", true),
        }
    }
}

# Role → Node
nodes contains node if {
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
		"labels": helpers.labels_to_list(resource),
		"annotations": helpers.annotations_to_list(resource),
		"labels_map": helpers.get_labels(resource),
		"annotations_map": helpers.get_annotations(resource),
		"perms": perms(rules),
		}
	}
}

# ClusterRole → Node
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "ClusterRole"
    rules := object.get(resource, "rules", [])

	node := {
		"id": helpers.generate_id("ClusterRole", "", resource.metadata.name),
		"kinds": ["ClusterRole"],
		"properties": {
			"name": helpers.get_name(resource),
			"resource_type": "clusterrole",
		"labels": helpers.labels_to_list(resource),
		"annotations": helpers.annotations_to_list(resource),
		"labels_map": helpers.get_labels(resource),
		"annotations_map": helpers.get_annotations(resource),
		"perms": perms(rules),
		}
	}
}

# RoleBinding → Node
nodes contains node if {
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
		"labels": helpers.labels_to_list(resource),
		"annotations": helpers.annotations_to_list(resource),
		"labels_map": helpers.get_labels(resource),
		"annotations_map": helpers.get_annotations(resource),
            "subjects": subjects,
            "role_ref": role_ref,
            "role_name": object.get(role_ref, "name", ""),
            "role_kind": object.get(role_ref, "kind", ""),
        }
    }
}

# ClusterRoleBinding → Node
nodes contains node if {
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
		"labels": helpers.labels_to_list(resource),
		"annotations": helpers.annotations_to_list(resource),
		"labels_map": helpers.get_labels(resource),
		"annotations_map": helpers.get_annotations(resource),
            "subjects": subjects,
            "role_ref": role_ref,
            "role_name": object.get(role_ref, "name", ""),
            "role_kind": object.get(role_ref, "kind", ""),
        }
    }
}
