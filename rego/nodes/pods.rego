# Pod Node Policies
# Creates BloodHound nodes for Kubernetes Pods with security analysis

package nodes.pods

import rego.v1
import data.nodes.base

# TODO: Remove all the excessive helper functions and just extract the relevant properties directly in the main node creation, as they are not reused anywhere else and are making the code more complex than it needs to be.
# Properties under __private are not included in the main node properties, so they can be any structure supported by rego, and are not limited to opengraph constraints.

# Main pod node creation
nodes contains node if {
	some i
	resource := input.resources[i]
	resource.kind == "Pod"
	metadata := base.extract_metadata(resource)
	
	sec := object.get(resource.spec, "securityContext", {})
	seccomp := object.get(sec, "seccompProfile", {})
	cap_add := extract_capabilities_add(resource.spec)
	cap_drop := extract_capabilities_drop(resource.spec)
	private := object.union(metadata.__private, {
		"containers": analyze_containers_detail(resource.spec),
		"volumes": analyze_volumes_detail(resource.spec),
		"capabilitiesAdd": cap_add,
		"capabilitiesDrop": cap_drop,
	})

	properties := object.union(metadata, {
		"__private": private,
		"securityContextConstraint": object.get(object.get(resource.metadata, "annotations", {}), "openshift.io/scc", "NotDefined"),
		"nodeName": object.get(resource.spec, "nodeName", ""),
		"containers": analyze_containers_summary(resource.spec),
		"containerImages": extract_container_images(resource.spec),
		"capabilitiesAdd": cap_add,
		"capabilitiesDrop": cap_drop,
		"hostNetwork": object.get(resource.spec, "hostNetwork", "NotDefined"),
		"hostPid": object.get(resource.spec, "hostPID", "NotDefined"),
		"hostIpc": object.get(resource.spec, "hostIPC", "NotDefined"),
		"runAsUser": object.get(sec, "runAsUser", "NotDefined"),
		"runAsGroup": object.get(sec, "runAsGroup", "NotDefined"),
		"runAsNonRoot": object.get(sec, "runAsNonRoot", "NotDefined"),
		"fsGroup": object.get(sec, "fsGroup", "NotDefined"),
		"supplementalGroups": object.get(sec, "supplementalGroups", []),
		"seccompProfile": object.get(seccomp, "type", "NotDefined"),
		"appArmorProfile": object.get(sec, "appArmorProfile", "NotDefined"),
		"seLinuxOptions": selinux_summary(sec),
		"volumes": analyze_volumes_summary(resource.spec),
		"serviceAccount": object.get(resource.spec, "serviceAccountName", "default"),
	})
	
node := base.default_node("pod", ["Pod"], metadata.namespace, metadata.name, properties)
}

# Extract container images
extract_container_images(spec) := images if {
	spec.containers
	images := [c.image | c := spec.containers[_]]
}

extract_container_images(spec) := [] if {
	not spec.containers
}

# Analyze containers (detail for edges)
analyze_containers_detail(spec) := containers if {
	spec.containers
	containers := [container |
		some i
		c := spec.containers[i]
		sec := object.get(c, "securityContext", {})
		container := {
			"name": c.name,
			"image": c.image,
			"privileged": object.get(sec, "privileged", "NotDefined"),
			"runAsUser": object.get(sec, "runAsUser", "NotDefined"),
			"runAsNonRoot": object.get(sec, "runAsNonRoot", "NotDefined"),
			"readOnlyRootFilesystem": object.get(sec, "readOnlyRootFilesystem", "NotDefined"),
			"envFrom": extract_env_from(c),
			"hasSecrets": references_secrets(c),
			"hostPorts": extract_host_ports(c),
			"volumeMounts": [m | m := c.volumeMounts[_]],
		}
	]
}

analyze_containers_detail(spec) := [] if {
	not spec.containers
}

# Analyze init containers (detail for edges)
analyze_init_containers_detail(spec) := containers if {
	spec.initContainers
	containers := [container |
		some i
		c := spec.initContainers[i]
		sec := object.get(c, "securityContext", {})
		container := {
			"name": c.name,
			"image": c.image,
			"privileged": object.get(sec, "privileged", "NotDefined"),
		}
	]
}

analyze_init_containers_detail(spec) := [] if {
	not spec.initContainers
}

# Analyze containers (summary for output)
analyze_containers_summary(spec) := summaries if {
	containers := analyze_containers_detail(spec)
	init := analyze_init_containers_detail(spec)
	container_summaries := [container_summary(c, "container") | c := containers[_]]
	init_summaries := [container_summary(c, "init") | c := init[_]]
	summaries := array.concat(container_summaries, init_summaries)
}

analyze_containers_summary(spec) := [] if {
	not spec.containers
	not spec.initContainers
}

container_summary(container, kind) := summary if {
	summary := sprintf("%s: image=%s, privileged=%v, runAsUser=%v, runAsNonRoot=%v, readOnlyRootFilesystem=%v, hasSecrets=%v", [
		kind_name(container.name, kind),
		container.image,
		container.privileged,
		container.runAsUser,
		container.runAsNonRoot,
		container.readOnlyRootFilesystem,
		container.hasSecrets,
	])
}

kind_name(name, kind) := out if {
	kind == "container"
	out := name
}

kind_name(name, kind) := out if {
	kind == "init"
	out := sprintf("init/%s", [name])
}


# Extract environment variable sources
extract_env_from(container) := env_sources if {
	container.envFrom
	env_sources := [source |
		some i
		ef := container.envFrom[i]
		source := {
			"secretRef": object.get(ef, "secretRef", null),
			"configMapRef": object.get(ef, "configMapRef", null),
		}
	]
}

extract_env_from(container) := [] if {
	not container.envFrom
}

# Extract host ports
extract_host_ports(container) := ports if {
	container.ports
	ports := [port |
		some i
		p := container.ports[i]
		port := {
			"containerPort": object.get(p, "containerPort", 0),
			"hostPort": object.get(p, "hostPort", 0),
			"hostIP": object.get(p, "hostIP", ""),
			"protocol": object.get(p, "protocol", "TCP"),
		}
		port.hostPort != 0
	]
}

extract_host_ports(container) := [] if {
	not container.ports
}

# Check if container references secrets
has_secret_ref(container) if {
	ef := container.envFrom[_]
	ef.secretRef
}

has_secret_ref(container) if {
	env := container.env[_]
	env.valueFrom.secretKeyRef
}

references_secrets(container) := true if {
	has_secret_ref(container)
} else := false

# Analyze volumes (detail for edges)
analyze_volumes_detail(spec) := volumes if {
	spec.volumes
	volumes := [volume |
		some i
		v := spec.volumes[i]
		volume := {
			"name": v.name,
			"type": volume_type(v),
			"secretName": object.get(object.get(v, "secret", {}), "secretName", ""),
			"configMapName": object.get(object.get(v, "configMap", {}), "name", ""),
			"pvcName": object.get(object.get(v, "persistentVolumeClaim", {}), "claimName", ""),
			"hostPath": object.get(object.get(v, "hostPath", {}), "path", ""),
		}
	]
}

analyze_volumes_detail(spec) := [] if {
	not spec.volumes
}

# Analyze volumes (summary for output)
analyze_volumes_summary(spec) := summaries if {
	volumes := analyze_volumes_detail(spec)
	summaries := [volume_summary(v) | v := volumes[_]]
}

analyze_volumes_summary(spec) := [] if {
	not spec.volumes
}

volume_summary(volume) := summary if {
	summary := sprintf("%s: type=%s, secret=%s, configMap=%s, pvc=%s, hostPath=%s", [
		volume.name,
		volume.type,
		volume.secretName,
		volume.configMapName,
		volume.pvcName,
		volume.hostPath,
	])
}

# Determine volume type
volume_type(volume) := "secret" if {
	volume.secret
} else := "configmap" if {
	volume.configMap
} else := "persistentVolumeClaim" if {
	volume.persistentVolumeClaim
} else := "hostPath" if {
	volume.hostPath
} else := "emptyDir" if {
	volume.emptyDir
} else := "projected" if {
	volume.projected
} else := "downwardAPI" if {
	volume.downwardAPI
} else := "other" 

# Analyze pod security context
analyze_pod_security(spec) := security if {
	spec.securityContext
	security := {
		"runAsUser": object.get(spec.securityContext, "runAsUser", "NotDefined"),
		"runAsGroup": object.get(spec.securityContext, "runAsGroup", "NotDefined"),
		"runAsNonRoot": object.get(spec.securityContext, "runAsNonRoot", "NotDefined"),
		"fsGroup": object.get(spec.securityContext, "fsGroup", "NotDefined"),
		"supplementalGroups": object.get(spec.securityContext, "supplementalGroups", []),
		"seccompProfile": object.get(spec.securityContext, "seccompProfile", "NotDefined"),
		"seLinuxOptions": object.get(spec.securityContext, "seLinuxOptions", "NotDefined"),
		"appArmorProfile": object.get(spec.securityContext, "appArmorProfile", "NotDefined"),
	}
}

analyze_pod_security(spec) := {} if {
	not spec.securityContext
}

selinux_summary(sec) := summary if {
	options := object.get(sec, "seLinuxOptions", {})
	count(object.keys(options)) > 0
	summary := sprintf("user=%v, role=%v, type=%v, level=%v", [
		object.get(options, "user", "NotDefined"),
		object.get(options, "role", "NotDefined"),
		object.get(options, "type", "NotDefined"),
		object.get(options, "level", "NotDefined"),
	])
}

selinux_summary(sec) := "NotDefined" if {
	options := object.get(sec, "seLinuxOptions", {})
	count(object.keys(options)) == 0
}

extract_capabilities_add(spec) := caps if {
	spec.containers
	list := [cap |
		c := spec.containers[_]
		cap := object.get(c, ["securityContext", "capabilities", "add"], [])[_]
	]
	caps := sort(uniq(list))
}

extract_capabilities_add(spec) := [] if {
	not spec.containers
}

extract_capabilities_drop(spec) := caps if {
	spec.containers
	list := [cap |
		c := spec.containers[_]
		cap := object.get(c, ["securityContext", "capabilities", "drop"], [])[_]
	]
	caps := sort(uniq(list))
}

extract_capabilities_drop(spec) := [] if {
	not spec.containers
}

uniq(list) := unique if {
	unique := {x | x := list[_]}
}
