# Generic Node Policies
# Creates nodes for Kubernetes resources without specific node policies

package nodes.generic

import rego.v1

import data.nodes.base
import data.nodes.helpers
import data.nodes.config

nodes contains node if {
    resource := input.resources[_]
    kind := object.get(resource, "kind", "")
    kind != ""
    not helpers.is_known_kind(kind)
    resource.metadata.name != ""

    metadata := base.extract_metadata(resource)
    api_version := object.get(resource, "apiVersion", "")
    resource_type := lower(kind)

    properties := object.union(metadata, {
        "resource_type": resource_type,
        "kind": kind,
        "apiVersion": api_version,
        "api_group": api_group(api_version),
        "api_version": api_version_value(api_version),
    })

    node := base.default_node(resource_type, [kind], metadata.namespace, metadata.name, properties)
}

api_group(api_version) := group if {
    contains(api_version, "/")
    parts := split(api_version, "/")
    group := parts[0]
}

api_group(api_version) := "" if {
    not contains(api_version, "/")
}

api_version_value(api_version) := version if {
    contains(api_version, "/")
    parts := split(api_version, "/")
    count(parts) > 1
    version := parts[1]
}

api_version_value(api_version) := api_version if {
    not contains(api_version, "/")
}
