# Pod Node Policies
# Creates BloodHound nodes for Kubernetes Pods with security analysis

package nodes.pods

import rego.v1
import data.nodes.base

# Main pod node creation
nodes contains node if {
	some i
	resource := input.resources[i]
	resource.kind == "Pod"
	metadata := base.extract_metadata(resource)
	
	properties := object.union(metadata, {
		"containers": analyze_containers(resource.spec),
		"init_containers": analyze_init_containers(resource.spec),
		"service_account": object.get(resource.spec, "serviceAccountName", "default"),
		"node_name": object.get(resource.spec, "nodeName", ""),
		"host_network": object.get(resource.spec, "hostNetwork", false),
		"host_pid": object.get(resource.spec, "hostPID", false),
		"host_ipc": object.get(resource.spec, "hostIPC", false),
		"security_context": analyze_pod_security(resource.spec),
		"volumes": analyze_volumes(resource.spec),
		"security_risk_score": calculate_security_risk(resource.spec),
		"privileged": is_privileged_pod(resource.spec),
		"runs_as_root": runs_as_root_pod(resource.spec),
	})
	
	node := base.default_node("pod", ["Pod"], metadata.namespace, metadata.name, properties)
}

# Analyze containers
analyze_containers(spec) := containers if {
	spec.containers
	containers := [container |
		some i
		c := spec.containers[i]
		container := {
			"name": c.name,
			"image": c.image,
			"privileged": object.get(c.securityContext, "privileged", false),
			"run_as_user": object.get(c.securityContext, "runAsUser", null),
			"run_as_non_root": object.get(c.securityContext, "runAsNonRoot", false),
			"read_only_root_filesystem": object.get(c.securityContext, "readOnlyRootFilesystem", false),
			"capabilities": extract_capabilities(c),
			"volume_mounts": extract_volume_mounts(c),
			"env_from": extract_env_from(c),
			"has_secrets": references_secrets(c),
		}
	]
}

analyze_containers(spec) := [] if {
	not spec.containers
}

analyze_init_containers(spec) := containers if {
	spec.initContainers
	containers := [container |
		some i
		c := spec.initContainers[i]
		container := {
			"name": c.name,
			"image": c.image,
			"privileged": object.get(c.securityContext, "privileged", false),
		}
	]
}

analyze_init_containers(spec) := [] if {
	not spec.initContainers
}

# Extract capabilities
extract_capabilities(container) := caps if {
	container.securityContext.capabilities
	caps := {
		"add": object.get(container.securityContext.capabilities, "add", []),
		"drop": object.get(container.securityContext.capabilities, "drop", []),
		"has_dangerous": has_dangerous_caps(container),
	}
}

extract_capabilities(container) := {} if {
	not container.securityContext.capabilities
}

has_dangerous_caps(container) if {
	some cap
	cap := container.securityContext.capabilities.add[_]
	cap in base.dangerous_capabilities
}

# Extract volume mounts
extract_volume_mounts(container) := mounts if {
	container.volumeMounts
	mounts := [mount |
		some i
		vm := container.volumeMounts[i]
		mount := {
			"name": vm.name,
			"mount_path": vm.mountPath,
			"read_only": object.get(vm, "readOnly", false),
			"sub_path": object.get(vm, "subPath", ""),
		}
	]
}

extract_volume_mounts(container) := [] if {
	not container.volumeMounts
}

# Extract environment variable sources
extract_env_from(container) := env_sources if {
	container.envFrom
	env_sources := [source |
		some i
		ef := container.envFrom[i]
		source := {
			"secret_ref": object.get(ef, "secretRef", null),
			"configmap_ref": object.get(ef, "configMapRef", null),
		}
	]
}

extract_env_from(container) := [] if {
	not container.envFrom
}

# Check if container references secrets
references_secrets(container) if {
	some ef
	ef := container.envFrom[_]
	ef.secretRef
}

references_secrets(container) if {
	some env
	env := container.env[_]
	env.valueFrom.secretKeyRef
}

# Analyze volumes
analyze_volumes(spec) := volumes if {
	spec.volumes
	volumes := [volume |
		some i
		v := spec.volumes[i]
		volume := {
			"name": v.name,
			"type": volume_type(v),
			"secret_name": object.get(object.get(v, "secret", {}), "secretName", ""),
			"configmap_name": object.get(object.get(v, "configMap", {}), "name", ""),
			"pvc_name": object.get(object.get(v, "persistentVolumeClaim", {}), "claimName", ""),
			"host_path": object.get(object.get(v, "hostPath", {}), "path", ""),
			"is_sensitive": is_sensitive_volume(v),
		}
	]
}

analyze_volumes(spec) := [] if {
	not spec.volumes
}

# Determine volume type
volume_type(volume) := "secret" if {
	volume.secret
}

volume_type(volume) := "configmap" if {
	volume.configMap
}

volume_type(volume) := "persistentVolumeClaim" if {
	volume.persistentVolumeClaim
}

volume_type(volume) := "hostPath" if {
	volume.hostPath
}

volume_type(volume) := "emptyDir" if {
	volume.emptyDir
}

volume_type(volume) := "projected" if {
	volume.projected
}

volume_type(volume) := "downwardAPI" if {
	volume.downwardAPI
}

volume_type(volume) := "other"

# Check if volume is sensitive
is_sensitive_volume(volume) if {
	volume.secret
}

is_sensitive_volume(volume) if {
	volume.hostPath
	sensitive_paths := ["/etc", "/var/run", "/proc", "/sys", "/dev"]
	some path
	path := sensitive_paths[_]
	startswith(volume.hostPath.path, path)
}

# Analyze pod security context
analyze_pod_security(spec) := security if {
	spec.securityContext
	security := {
		"run_as_user": object.get(spec.securityContext, "runAsUser", null),
		"run_as_group": object.get(spec.securityContext, "runAsGroup", null),
		"run_as_non_root": object.get(spec.securityContext, "runAsNonRoot", false),
		"fs_group": object.get(spec.securityContext, "fsGroup", null),
		"supplemental_groups": object.get(spec.securityContext, "supplementalGroups", []),
		"seccomp_profile": object.get(spec.securityContext, "seccompProfile", null),
		"se_linux_options": object.get(spec.securityContext, "seLinuxOptions", null),
	}
}

analyze_pod_security(spec) := {} if {
	not spec.securityContext
}

# Calculate security risk score (0-100)
calculate_security_risk(spec) := score if {
	risk_factors := [
		privileged_risk(spec),
		host_namespace_risk(spec),
		capabilities_risk(spec),
		root_risk(spec),
		volume_risk(spec),
	]
	score := sum(risk_factors)
}

privileged_risk(spec) := 30 if {
	is_privileged_pod(spec)
}

privileged_risk(spec) := 0

host_namespace_risk(spec) := 20 if {
	spec.hostNetwork == true
}

host_namespace_risk(spec) := 15 if {
	spec.hostPID == true
}

host_namespace_risk(spec) := 10 if {
	spec.hostIPC == true
}

host_namespace_risk(spec) := 0

capabilities_risk(spec) := 25 if {
	some container
	container := spec.containers[_]
	base.has_dangerous_capabilities(container)
}

capabilities_risk(spec) := 0

root_risk(spec) := 15 if {
	runs_as_root_pod(spec)
}

root_risk(spec) := 0

volume_risk(spec) := 20 if {
	some volume
	volume := spec.volumes[_]
	is_sensitive_volume(volume)
}

volume_risk(spec) := 0

# Helper: Check if pod is privileged
is_privileged_pod(spec) if {
	some container
	container := spec.containers[_]
	base.is_privileged(container)
}

is_privileged_pod(spec) := false if {
	not is_privileged_pod(spec)
}

# Helper: Check if pod runs as root
runs_as_root_pod(spec) if {
	some container
	container := spec.containers[_]
	base.runs_as_root(container)
}

runs_as_root_pod(spec) := false if {
	not runs_as_root_pod(spec)
}