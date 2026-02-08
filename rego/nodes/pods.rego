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
		sec := object.get(c, "securityContext", {})
		container := {
			"name": c.name,
			"image": c.image,
			"privileged": object.get(sec, "privileged", false),
			"run_as_user": object.get(sec, "runAsUser", null),
			"run_as_non_root": object.get(sec, "runAsNonRoot", false),
			"read_only_root_filesystem": object.get(sec, "readOnlyRootFilesystem", false),
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
		sec := object.get(c, "securityContext", {})
		container := {
			"name": c.name,
			"image": c.image,
			"privileged": object.get(sec, "privileged", false),
		}
	]
}

analyze_init_containers(spec) := [] if {
	not spec.initContainers
}

# Extract capabilities
extract_capabilities(container) := caps if {
	sec := object.get(container, "securityContext", {})
	sec.capabilities
	caps := {
		"add": object.get(sec.capabilities, "add", []),
		"drop": object.get(sec.capabilities, "drop", []),
		"has_dangerous": has_dangerous_caps(container),
	}
}

extract_capabilities(container) := {} if {
	sec := object.get(container, "securityContext", {})
	not sec.capabilities
}

has_dangerous_caps(container) if {
	sec := object.get(container, "securityContext", {})
	cap := object.get(sec, ["capabilities", "add"], [])[_]
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

# Check if volume is sensitive
is_sensitive_volume(volume) if {
	volume.secret
}

is_sensitive_volume(volume) if {
	volume.hostPath
	sensitive_paths := ["/etc", "/var/run", "/proc", "/sys", "/dev"]
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
} else := 0

host_namespace_risk(spec) := 20 if {
	spec.hostNetwork == true
} else := 15 if {
	spec.hostPID == true
} else := 10 if {
	spec.hostIPC == true
} else := 0

capabilities_risk(spec) := 25 if {
	container := spec.containers[_]
	base.has_dangerous_capabilities(container)
} else := 0

root_risk(spec) := 15 if {
	runs_as_root_pod(spec)
} else := 0

volume_risk(spec) := 20 if {
	volume := spec.volumes[_]
	is_sensitive_volume(volume)
} else := 0

# Helper: Check if pod is privileged
has_privileged_container(spec) if {
	container := spec.containers[_]
	base.is_privileged(container)
}

is_privileged_pod(spec) := true if {
	has_privileged_container(spec)
} else := false

# Helper: Check if pod runs as root
has_root_container(spec) if {
	container := spec.containers[_]
	base.runs_as_root(container)
}

runs_as_root_pod(spec) := true if {
	has_root_container(spec)
} else := false
