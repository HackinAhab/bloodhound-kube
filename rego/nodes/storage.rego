package nodes.storage

import rego.v1

import data.nodes.helpers

# Secret → Node
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "Secret"
    
    data_keys := object.get(resource, "data", {})
    keys := object.keys(data_keys)
    
    node := {
        "id": helpers.generate_id("Secret", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["Secret"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "secret",
            "secret_type": object.get(resource, "type", "Opaque"),
            "data_keys": keys,
            "data_keys_count": count(keys),
            "has_sensitive_keys": helpers.has_sensitive_keys(keys),
            "certificates": object.get(resource, "certificates", {}),
            "redacted_keys": object.get(resource, "redacted_keys", []),
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "is_service_account_token": object.get(resource, "type", "") == "kubernetes.io/service-account-token",
            "is_tls_secret": object.get(resource, "type", "") == "kubernetes.io/tls",
            "is_opaque": object.get(resource, "type", "Opaque") == "Opaque",
        }
    }
}

# ConfigMap → Node
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "ConfigMap"
    
    data_keys := object.get(resource, "data", {})
    keys := object.keys(data_keys)
    
    node := {
        "id": helpers.generate_id("ConfigMap", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["ConfigMap"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "configmap",
            "data_keys": keys,
            "data_keys_count": count(keys),
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
        }
    }
}
