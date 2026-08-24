//go:build !no_addons && !no_cert_manager

package certmanager

import (
	. "bloodhound-kube/internal/nodes/framework"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register() {
	RegisterResources(
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"},
			MapBuilder: BuildCertificateNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Issuer"},
			MapBuilder: BuildIssuerNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "ClusterIssuer"},
			MapBuilder: BuildClusterIssuerNode,
			FetchMode:  FetchModeHintFull,
		},
	)
	RegisterDefaultCollections(
		CollectionTarget{
			Group:    "cert-manager.io",
			Version:  "v1",
			Resource: "certificates",
		},
		CollectionTarget{
			Group:    "cert-manager.io",
			Version:  "v1",
			Resource: "issuers",
		},
		CollectionTarget{
			Group:    "cert-manager.io",
			Version:  "v1",
			Resource: "clusterissuers",
		},
	)
}
