# External Secrets Node Policies
# Creates BloodHound nodes for external-secrets.io resources

package nodes.external_secrets

import rego.v1

import data.nodes.helpers

# SecretStore -> Node (namespaced)
nodes contains node if {
    resource := input.resources[_]
    lower(resource.kind) == "secretstore"

    provider := object.get(resource, ["spec", "provider"], {})
    provider_keys := helpers.get_keys(provider)
    provider_key_list := sort(provider_keys)
    count(provider_key_list) > 0
    provider_type := provider_key_list[0]

    node := {
        "id": helpers.generate_id("SecretStore", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["SecretStore"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "secretstore",
            "labels": helpers.labels_to_list(resource),
            "annotations": helpers.annotations_to_list(resource),
            "labels_map": helpers.get_labels(resource),
            "annotations_map": helpers.get_annotations(resource),
            "provider_type": provider_type,
        }
    }
}

nodes contains node if {
    resource := input.resources[_]
    lower(resource.kind) == "secretstore"

    provider := object.get(resource, ["spec", "provider"], {})
    provider_keys := helpers.get_keys(provider)
    provider_key_list := sort(provider_keys)
    count(provider_key_list) == 0

    node := {
        "id": helpers.generate_id("SecretStore", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["SecretStore"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "secretstore",
            "labels": helpers.labels_to_list(resource),
            "annotations": helpers.annotations_to_list(resource),
            "labels_map": helpers.get_labels(resource),
            "annotations_map": helpers.get_annotations(resource),
            "provider_type": "",
        }
    }
}

# ClusterSecretStore -> Node (cluster-scoped)
nodes contains node if {
    resource := input.resources[_]
    lower(resource.kind) == "clustersecretstore"

    provider := object.get(resource, ["spec", "provider"], {})
    provider_keys := helpers.get_keys(provider)
    provider_key_list := sort(provider_keys)
    count(provider_key_list) > 0
    provider_type := provider_key_list[0]

    node := {
        "id": helpers.generate_id("ClusterSecretStore", "", resource.metadata.name),
        "kinds": ["ClusterSecretStore"],
        "properties": {
            "name": helpers.get_name(resource),
            "resource_type": "clustersecretstore",
            "labels": helpers.labels_to_list(resource),
            "annotations": helpers.annotations_to_list(resource),
            "labels_map": helpers.get_labels(resource),
            "annotations_map": helpers.get_annotations(resource),
            "provider_type": provider_type,
        }
    }
}

nodes contains node if {
    resource := input.resources[_]
    lower(resource.kind) == "clustersecretstore"

    provider := object.get(resource, ["spec", "provider"], {})
    provider_keys := helpers.get_keys(provider)
    provider_key_list := sort(provider_keys)
    count(provider_key_list) == 0

    node := {
        "id": helpers.generate_id("ClusterSecretStore", "", resource.metadata.name),
        "kinds": ["ClusterSecretStore"],
        "properties": {
            "name": helpers.get_name(resource),
            "resource_type": "clustersecretstore",
            "labels": helpers.labels_to_list(resource),
            "annotations": helpers.annotations_to_list(resource),
            "labels_map": helpers.get_labels(resource),
            "annotations_map": helpers.get_annotations(resource),
            "provider_type": "",
        }
    }
}

# ExternalSecret -> Node (namespaced)
nodes contains node if {
    resource := input.resources[_]
    lower(resource.kind) == "externalsecret"

    data_keys := [key |
        item := object.get(resource, ["spec", "data"], [])[_]
        key := object.get(item, "secretKey", "")
        key != ""
    ]

    data_from_types := [source_type |
        item := object.get(resource, ["spec", "dataFrom"], [])[_]
        keys := helpers.get_keys(item)
        key_list := sort(keys)
        count(key_list) > 0
        source_type := key_list[0]
    ]

    store_ref := object.get(resource, ["spec", "secretStoreRef"], {})

    node := {
        "id": helpers.generate_id("ExternalSecret", resource.metadata.namespace, resource.metadata.name),
        "kinds": ["ExternalSecret"],
        "properties": {
            "name": helpers.get_name(resource),
            "namespace": helpers.get_namespace(resource),
            "resource_type": "externalsecret",
            "labels": helpers.labels_to_list(resource),
            "annotations": helpers.annotations_to_list(resource),
            "labels_map": helpers.get_labels(resource),
            "annotations_map": helpers.get_annotations(resource),
            "store_name": object.get(store_ref, "name", ""),
            "store_kind": object.get(store_ref, "kind", "SecretStore"),
            "target_name": object.get(object.get(resource, ["spec", "target"], {}), "name", ""),
            "refresh_interval": object.get(resource, ["spec", "refreshInterval"], ""),
            "creation_policy": object.get(object.get(resource, ["spec", "target"], {}), "creationPolicy", ""),
            "data_keys": data_keys,
            "data_from_types": data_from_types,
        }
    }
}
