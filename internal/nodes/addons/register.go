package addons

import (
	"bloodhound-kube/internal/nodes/framework"

	securityv1 "github.com/openshift/api/security/v1"
	apiv3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register(reg *framework.Registry) {
	reg.Register("SecretStore", BuildSecretStoreNode)
	reg.Register("ClusterSecretStore", BuildClusterSecretStoreNode)
	reg.Register("ExternalSecret", BuildExternalSecretNode)
	reg.RegisterTyped(securityv1.SchemeGroupVersion.WithKind("SecurityContextConstraints"), BuildSecurityContextConstraintsNode)

	reg.RegisterTypedWithFetchMode(apiv3.SchemeGroupVersion.WithKind("GlobalNetworkPolicy"), BuildGlobalNetworkPolicyNode, framework.FetchModeHintFull)
	reg.RegisterTypedFromMapWithFetchMode(schema.GroupVersionKind{Group: "cilium.io", Version: "v2", Kind: "CiliumNetworkPolicy"}, BuildCiliumNetworkPolicyNode, framework.FetchModeHintFull)
}
