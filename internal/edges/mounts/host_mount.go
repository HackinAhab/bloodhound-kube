package mounts

import "strings"

var HostMountReadEdgeRule = hostMountRule{
	name:           "lateral_movement_host_mount_read",
	edgeKind:       "BHK_hostMountSensitive",
	sensitivePaths: []string{"/etc", "/root", "/home", "/proc", "/var/lib/kubelet/pods", "/var/run", "/sys", "/dev", "/run", "/usr"},
	exclude:        func(hostPath string) bool { return strings.HasSuffix(hostPath, ".sock") },
	props: map[string]any{
		"Description": "Pod has a host mount of sensitive directories, which can allow for lateral movement and potential information disclosure",
		"Reference":   "https://kubehound.io/reference/attacks/EXPLOIT_HOST_READ/",
	},
}
