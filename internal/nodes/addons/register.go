package addons

import "bloodhound-kube/internal/nodes/framework"

func Register(reg *framework.Registry) {
	reg.Register("SecretStore", BuildSecretStoreNode)
	reg.Register("ClusterSecretStore", BuildClusterSecretStoreNode)
	reg.Register("ExternalSecret", BuildExternalSecretNode)
}
