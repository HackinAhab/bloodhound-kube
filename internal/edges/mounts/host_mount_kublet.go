package mounts

var HostMountKubeletEdgeRule = hostMountRule{
	name:           "lateral_movement_host_mount_kubelet",
	edgeKind:       "BHK_mountedKubelet",
	sensitivePaths: []string{"/var/lib/kubelet", "/etc/kubernetes"},
	props: map[string]any{
		"Description": "Pod has a host mount containing a common kubelet directory, which may allow access to the node kubelet configuration and credentials. This check is left relatively broad to catch various kubelet-related host mounts, but may produce false positives and duplicates with LM_HOST_MOUNT_READ if common kubelet subdirectories are included in the host mount path.",
		"Reference":   "",
	},
}
