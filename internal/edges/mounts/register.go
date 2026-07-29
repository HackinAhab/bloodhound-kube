package mounts

import "bloodhound-kube/internal/edges/framework"

func Register(reg *framework.Registry) {
	reg.Register(HostMountReadEdgeRule)
	reg.Register(HostMountKubeletEdgeRule)
	reg.Register(PodMountServiceAccountEdgeRule{})
	reg.Register(PersistentVolumesEdgesRule{})
}
