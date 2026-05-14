package addons

import "bloodhound-kube/internal/edges/framework"

func Register(reg *framework.Registry) {
	reg.Register(externalSecretsEdgesRule{})
}
