# Deployment Node Policies
# Creates BloodHound nodes for Kubernetes Deployments

package nodes.deployments

import rego.v1
import data.nodes.base
import data.nodes.helpers

# Main deployment node creation
nodes contains node if {
	some i
	resource := input.resources[i]
	resource.kind == "Deployment"
	metadata := base.extract_metadata(resource)
	
	selector_map := object.get(resource.spec.selector, "matchLabels", {})
	private := object.union(metadata.__private, {
		"selector_map": selector_map,
	})
	properties := object.union(metadata, {
		"replicas": object.get(resource.spec, "replicas", 1),
		"selector": helpers.labels_map_to_list(selector_map),
		"__private": private,
		"strategy_type": object.get(resource.spec.strategy, "type", "RollingUpdate"),
		"pod_template": analyze_pod_template(resource.spec.template),
	})
	
	node := base.default_node("deployment", ["Deployment"], metadata.namespace, metadata.name, properties)
}

# Analyze pod template
analyze_pod_template(template) := pod_info if {
	labels_map := object.get(template.metadata, "labels", {})
	annotations_map := object.get(template.metadata, "annotations", {})
	pod_info := {
		"labels": helpers.labels_map_to_list(labels_map),
		"annotations": helpers.annotations_map_to_list(annotations_map),
		"__private": {
			"labels_map": labels_map,
			"annotations_map": annotations_map,
		},
		"serviceAccount": object.get(template.spec, "serviceAccountName", "default"),
		"containers": extract_container_info(template.spec),
		"containerImages": extract_container_images(template.spec),
		"volumes": extract_volume_info(template.spec),
		"securityContext": object.get(template.spec, "securityContext", {}),
	}
}

# Extract container images
extract_container_images(spec) := images if {
	spec.containers
	images := [c.image | c := spec.containers[_]]
}

extract_container_images(spec) := [] if {
	not spec.containers
}

# Extract container information
extract_container_info(spec) := containers if {
	spec.containers
	containers := [container |
		some i
		c := spec.containers[i]
		container := {
			"name": c.name,
			"image": c.image,
			"pullPolicy": object.get(c, "imagePullPolicy", "IfNotPresent"),
			"envFromSecrets": has_env_from_secrets(c),
			"envFromConfigMaps": has_env_from_configmaps(c),
		}
	]
}

extract_container_info(spec) := [] if {
	not spec.containers
}

# Check if container has env from secrets
has_env_from_secrets(container) if {
	ef := container.envFrom[_]
	ef.secretRef
}

has_env_from_secrets(container) if {
	env := container.env[_]
	env.valueFrom.secretKeyRef
}

# Check if container has env from configmaps
has_env_from_configmaps(container) if {
	ef := container.envFrom[_]
	ef.configMapRef
}

has_env_from_configmaps(container) if {
	env := container.env[_]
	env.valueFrom.configMapKeyRef
}

# Extract volume information
extract_volume_info(spec) := volumes if {
	spec.volumes
	volumes := [volume |
		some i
		v := spec.volumes[i]
		volume := {
			"name": v.name,
			"type": determine_volume_type(v),
		}
	]
}

extract_volume_info(spec) := [] if {
	not spec.volumes
}

# Determine volume type
determine_volume_type(volume) := "secret" if {
	volume.secret
} else := "configMap" if {
	volume.configMap
} else := "persistentVolumeClaim" if {
	volume.persistentVolumeClaim
} else := "emptyDir" if {
	volume.emptyDir
} else := "hostPath" if {
	volume.hostPath
} else := "other"
