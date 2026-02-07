# External Secrets Node Policies
# Creates BloodHound nodes for external-secrets.io resources

package nodes.external_secrets

import rego.v1

import data.nodes.helpers

# SecretStore -> Node (namespaced)
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "SecretStore"

    provider_keys := helpers.get_keys(object.get(resource.spec, "provider", {}))
    count(provider_keys) > 0
    provider_type := provider_keys[0]

    node := {
        "id": helpers.generate_id("SecretStore", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["SecretStore"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "secretstore",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "provider_type": provider_type,
        }
    }
}

nodes contains node if {
    resource := input.resources[_]
    resource.kind == "SecretStore"

    provider_keys := helpers.get_keys(object.get(resource.spec, "provider", {}))
    count(provider_keys) == 0

    node := {
        "id": helpers.generate_id("SecretStore", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["SecretStore"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "secretstore",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "provider_type": "",
        }
    }
}

# ClusterSecretStore -> Node (cluster-scoped)
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "ClusterSecretStore"

    provider_keys := helpers.get_keys(object.get(resource.spec, "provider", {}))
    count(provider_keys) > 0
    provider_type := provider_keys[0]

    node := {
        "id": helpers.generate_id("ClusterSecretStore", "", resource.metadata.name),
        "kinds": ["ClusterSecretStore"],
        "properties": {
            "name": helpers.get_name(resource),
            "resource_type": "clustersecretstore",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "provider_type": provider_type,
        }
    }
}

nodes contains node if {
    resource := input.resources[_]
    resource.kind == "ClusterSecretStore"

    provider_keys := helpers.get_keys(object.get(resource.spec, "provider", {}))
    count(provider_keys) == 0

    node := {
        "id": helpers.generate_id("ClusterSecretStore", "", resource.metadata.name),
        "kinds": ["ClusterSecretStore"],
        "properties": {
            "name": helpers.get_name(resource),
            "resource_type": "clustersecretstore",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "provider_type": "",
        }
    }
}

# ExternalSecret -> Node (namespaced)
nodes contains node if {
    resource := input.resources[_]
    resource.kind == "ExternalSecret"

    data_keys := [key |
        item := object.get(resource.spec, "data", [])[_]
        key := object.get(item, "secretKey", "")
        key != ""
    ]

    data_from_types := [source_type |
        item := object.get(resource.spec, "dataFrom", [])[_]
        keys := helpers.get_keys(item)
        count(keys) > 0
        source_type := keys[0]
    ]

    store_ref := object.get(resource.spec, "secretStoreRef", {})

    node := {
        "id": helpers.generate_id("ExternalSecret", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["ExternalSecret"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "externalsecret",
            "labels": helpers.get_labels(resource),
            "annotations": helpers.get_annotations(resource),
            "store_name": object.get(store_ref, "name", ""),
            "store_kind": object.get(store_ref, "kind", "SecretStore"),
            "target_name": object.get(object.get(resource.spec, "target", {}), "name", ""),
            "refresh_interval": object.get(resource.spec, "refreshInterval", ""),
            "creation_policy": object.get(object.get(resource.spec, "target", {}), "creationPolicy", ""),
            "data_keys": data_keys,
            "data_from_types": data_from_types,
        }
    }
}
