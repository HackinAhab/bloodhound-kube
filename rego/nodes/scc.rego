package nodes.scc

import rego.v1

import data.nodes.helpers

# SecurityContextConstraints → Node
nodes contains node if {
	resource := input.resources[_]
	resource.kind == "SecurityContextConstraints"

	node := {
		"id": helpers.generate_id("SecurityContextConstraints", "", resource.metadata.name),
		"kinds": ["SecurityContextConstraints"],
		"properties": {
			"name": helpers.get_name(resource),
			"resource_type": "securitycontextconstraints",
			"labels": helpers.labels_to_list(resource),
			"annotations": helpers.annotations_to_list(resource),
			"__private": {
				"labels_map": helpers.get_labels(resource),
				"annotations_map": helpers.get_annotations(resource),
			},
		}
	}
}
