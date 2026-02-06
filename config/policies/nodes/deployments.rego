# Deployment Node Policies
# Creates BloodHound nodes for Kubernetes Deployments

package nodes.deployments

import rego.v1
import data.nodes.base

# Main deployment node creation
nodes contains node if {
	some i
	resource := input.resources[i]
	resource.kind == "Deployment"
	metadata := base.extract_metadata(resource)
	
	properties := object.union(metadata, {
		"replicas": object.get(resource.spec, "replicas", 1),
		"selector": object.get(resource.spec.selector, "matchLabels", {}),
		"strategy_type": object.get(resource.spec.strategy, "type", "RollingUpdate"),
		"pod_template": analyze_pod_template(resource.spec.template),
		"min_ready_seconds": object.get(resource.spec, "minReadySeconds", 0),
		"revision_history_limit": object.get(resource.spec, "revisionHistoryLimit", 10),
	})
	
	node := base.default_node("deployment", ["Deployment"], metadata.namespace, metadata.name, properties)
}

# Analyze pod template
analyze_pod_template(template) := pod_info if {
	pod_info := {
		"labels": object.get(template.metadata, "labels", {}),
		"annotations": object.get(template.metadata, "annotations", {}),
		"service_account": object.get(template.spec, "serviceAccountName", "default"),
		"containers": extract_container_info(template.spec),
		"volumes": extract_volume_info(template.spec),
		"security_context": object.get(template.spec, "securityContext", {}),
	}
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
			"pull_policy": object.get(c, "imagePullPolicy", "IfNotPresent"),
			"env_from_secrets": has_env_from_secrets(c),
			"env_from_configmaps": has_env_from_configmaps(c),
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
