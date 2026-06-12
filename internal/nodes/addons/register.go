package addons

import (
	"bloodhound-kube/internal/nodes/framework"

	securityv1 "github.com/openshift/api/security/v1"
)

func Register(reg *framework.Registry) {
	reg.Register("SecretStore", BuildSecretStoreNode)
	reg.Register("ClusterSecretStore", BuildClusterSecretStoreNode)
	reg.Register("ExternalSecret", BuildExternalSecretNode)
	reg.RegisterTyped(securityv1.SchemeGroupVersion.WithKind("SecurityContextConstraints"), BuildSecurityContextConstraintsNode)
}
