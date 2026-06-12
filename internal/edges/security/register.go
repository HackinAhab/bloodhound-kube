package security

import "bloodhound-kube/internal/edges/framework"

func Register(reg *framework.Registry) {
	reg.Register(sccEdgesRule{})
	reg.Register(hostPortsEdgesRule{})
	reg.Register(capabilityEdgesRule{})
	reg.Register(containerEscapeRule{})
}
