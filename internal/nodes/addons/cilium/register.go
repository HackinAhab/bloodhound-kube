//go:build !no_addons && !no_cilium

package cilium

import (
	. "bloodhound-kube/internal/nodes/framework"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register() {
	RegisterResources(
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "cilium.io", Version: "v2", Kind: "CiliumNetworkPolicy"},
			MapBuilder: BuildCiliumNetworkPolicyNode,
			FetchMode:  FetchModeHintFull,
		},
	)
	RegisterDefaultCollections(
		CollectionTarget{
			Group:    "cilium.io",
			Version:  "v2",
			Resource: "ciliumnetworkpolicies",
		},
	)
}
