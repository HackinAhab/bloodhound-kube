package nodes.httproutes

import rego.v1

import data.nodes.base

# HTTPRoute → Node
nodes contains node if {
	resource := input.resources[_]
	resource.kind == "HTTPRoute"
	metadata := base.extract_metadata(resource)

	hostnames := object.get(resource.spec, "hostnames", [])
	parent_refs := object.get(resource.spec, "parentRefs", [])
	backend_ref_keys := [key |
		rule := object.get(resource.spec, "rules", [])[_]
		backend := object.get(rule, "backendRefs", [])[_]
		object.get(backend, "kind", "Service") == "Service"
		backend.name != ""
		ns := object.get(backend, "namespace", "")
		key := sprintf("%s/%s", [ns, backend.name])
		key != "/"
	]

	properties := object.union(metadata, {
		"hostnames": hostnames,
		"parentRefs": parent_refs,
		"backendRefKeys": backend_ref_keys,
		"backendRefsCount": count(backend_ref_keys),
	})

	node := base.default_node("httproute", ["HTTPRoute"], metadata.namespace, metadata.name, properties)
}
