//go:build !no_addons && !no_cert_manager

package certmanager

import (
	"strings"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/workload"
)

func Register(reg *framework.Registry) {
	reg.Register(certManagerEdgesRule{})
}

type certManagerEdgesRule struct{}

func (r certManagerEdgesRule) Name() string { return "cert_manager" }

func (r certManagerEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		issuers := ctx.Index.IssuersByNamespace[ns]
		secrets := ctx.Index.SecretsByNamespace[ns]

		for i := range space.Certificates {
			cert := &space.Certificates[i]

			if cert.SecretName != "" {
				if secret := secrets[cert.SecretName]; secret != nil {
					edges = append(edges, framework.CreateEdge(cert, secret, "BHK_ManagedBy"))
				}
			}

			if cert.IssuerRefName == "" {
				continue
			}
			switch strings.ToLower(cert.IssuerRefKind) {
			case "clusterissuer":
				if issuer := ctx.Index.ClusterIssuersByName[cert.IssuerRefName]; issuer != nil {
					edges = append(edges, framework.CreateEdge(cert, issuer, "BHK_ManagedBy"))
				}
			default:
				if issuer := issuers[cert.IssuerRefName]; issuer != nil {
					edges = append(edges, framework.CreateEdge(cert, issuer, "BHK_ManagedBy"))
				}
			}
		}

		for i := range space.Issuers {
			issuer := &space.Issuers[i]
			edges = append(edges, secretRefEdges(issuer, issuer.CASecretName, issuer.VaultSecretName, secrets)...)
		}
	}

	// ClusterIssuer secretRefs have no namespace on the object itself (the
	// CRD is cluster-scoped); resolve against every namespace's secrets and
	// let the name match narrow it down, same tradeoff externalsecrets makes
	// for ClusterSecretStore lookups.
	for i := range ctx.Core.Cluster.ClusterIssuers {
		issuer := &ctx.Core.Cluster.ClusterIssuers[i]
		for _, secrets := range ctx.Index.SecretsByNamespace {
			edges = append(edges, secretRefEdges(issuer, issuer.CASecretName, issuer.VaultSecretName, secrets)...)
		}
	}
	return edges
}

// secretRefEdges emits BHK_ManagedBy edges from an Issuer/ClusterIssuer to
// its CA signing-key secret and/or Vault auth secret, if present in secrets.
func secretRefEdges(issuer nodefw.EdgeNode, caSecretName, vaultSecretName string, secrets map[string]*workload.Secret) []model.BloodHoundEdge {
	var edges []model.BloodHoundEdge
	if caSecretName != "" {
		if secret := secrets[caSecretName]; secret != nil {
			edges = append(edges, framework.CreateEdge(issuer, secret, "BHK_ManagedBy"))
		}
	}
	if vaultSecretName != "" {
		if secret := secrets[vaultSecretName]; secret != nil {
			edges = append(edges, framework.CreateEdge(issuer, secret, "BHK_ManagedBy"))
		}
	}
	return edges
}
