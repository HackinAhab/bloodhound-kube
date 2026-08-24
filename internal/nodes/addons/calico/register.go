//go:build !no_addons && !no_calico

package calico

import (
	. "bloodhound-kube/internal/nodes/framework"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register() {
	RegisterResources(
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "crd.projectcalico.org", Version: "v1", Kind: "GlobalNetworkPolicy"},
			MapBuilder: BuildGlobalNetworkPolicyMapNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "crd.projectcalico.org", Version: "v1", Kind: "HostEndpoint"},
			MapBuilder: BuildHostEndpointMapNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "GlobalNetworkPolicy"},
			MapBuilder: BuildGlobalNetworkPolicyMapNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "projectcalico.org", Version: "v3", Kind: "HostEndpoint"},
			MapBuilder: BuildHostEndpointMapNode,
			FetchMode:  FetchModeHintFull,
		},
	)
	RegisterDefaultCollections(
		CollectionTarget{
			Group:    "crd.projectcalico.org",
			Version:  "v1",
			Resource: "globalnetworkpolicies",
		},
		CollectionTarget{
			Group:    "crd.projectcalico.org",
			Version:  "v1",
			Resource: "hostendpoints",
		},
		CollectionTarget{
			Group:    "projectcalico.org",
			Version:  "v3",
			Resource: "globalnetworkpolicies",
		},
		CollectionTarget{
			Group:    "projectcalico.org",
			Version:  "v3",
			Resource: "hostendpoints",
		},
	)
}
