package certmanager

import . "bloodhound-kube/internal/nodes/framework"

// cert-manager type structs live here (untagged) because internal/model
// embeds them in CoreFacts. The parse/build logic is gated in
// cert_manager.go (//go:build !no_addons && !no_cert_manager).

type Certificate struct {
	GraphNodeBase
	SecretName    string
	IssuerRefName string
	IssuerRefKind string
}

type Issuer struct {
	GraphNodeBase
	CASecretName    string
	VaultSecretName string
}

type ClusterIssuer struct {
	GraphNodeBase
	CASecretName    string
	VaultSecretName string
}
