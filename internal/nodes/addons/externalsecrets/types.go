package externalsecrets

import . "bloodhound-kube/internal/nodes/framework"

// external-secrets type structs live here (untagged) because internal/model
// embeds them in CoreFacts. The parse/build logic is gated in
// external_secrets.go (//go:build !no_addons && !no_external_secrets).

type SecretStore struct {
	GraphNodeBase
	ProviderType string
}

type ClusterSecretStore struct {
	GraphNodeBase
	ProviderType string
}

type ExternalSecret struct {
	GraphNodeBase
	StoreName  string
	StoreKind  string
	TargetName string
}
