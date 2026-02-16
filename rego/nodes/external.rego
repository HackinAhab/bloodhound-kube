# External Node Policy
# Creates a single External node for out-of-cluster paths

package nodes.external

import rego.v1

import data.nodes.helpers

nodes contains node if {
	node := {
		"id": helpers.generate_id("External", "", "external"),
		"kinds": ["External"],
		"properties": {
			"name": "external",
			"resource_type": "external",
		},
	}
}
