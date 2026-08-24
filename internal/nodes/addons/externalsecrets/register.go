//go:build !no_addons && !no_external_secrets

package externalsecrets

import (
	. "bloodhound-kube/internal/nodes/framework"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Register() {
	RegisterResources(
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1", Kind: "SecretStore"},
			MapBuilder: BuildSecretStoreNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1", Kind: "ClusterSecretStore"},
			MapBuilder: BuildClusterSecretStoreNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1", Kind: "ExternalSecret"},
			MapBuilder: BuildExternalSecretNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1beta1", Kind: "SecretStore"},
			MapBuilder: BuildSecretStoreNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1beta1", Kind: "ClusterSecretStore"},
			MapBuilder: BuildClusterSecretStoreNode,
			FetchMode:  FetchModeHintFull,
		},
		ResourceRegistration{
			GVK:        schema.GroupVersionKind{Group: "external-secrets.io", Version: "v1beta1", Kind: "ExternalSecret"},
			MapBuilder: BuildExternalSecretNode,
			FetchMode:  FetchModeHintFull,
		},
	)
}
