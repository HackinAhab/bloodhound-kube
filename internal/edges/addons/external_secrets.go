package addons

import (
	"strings"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
)

type externalSecretsEdgesRule struct{}

func (r externalSecretsEdgesRule) Name() string { return "external_secrets" }

func (r externalSecretsEdgesRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		secretStores := ctx.Index.SecretStoresByNamespace[ns]
		for i := range space.ExternalSecrets {
			es := &space.ExternalSecrets[i]
			if es.StoreName != "" {
				storeKind := strings.ToLower(es.StoreKind)
				if storeKind == "" {
					storeKind = "secretstore"
				}
				if storeKind == "secretstore" {
					if store := secretStores[es.StoreName]; store != nil {
						edges = append(edges, framework.CreateEdge(es, store, "ManagedBy"))
					}
				}
				if storeKind == "clustersecretstore" {
					if store := ctx.Index.ClusterSecretStoresByName[es.StoreName]; store != nil {
						edges = append(edges, framework.CreateEdge(es, store, "ManagedBy"))
					}
				}
			}

			for j := range space.Secrets {
				secret := &space.Secrets[j]
				if es.TargetName != "" {
					if secret.Name == es.TargetName {
						edges = append(edges, framework.CreateEdge(secret, es, "ManagedBy"))
					}
					continue
				}
				if secret.Name == es.Name {
					edges = append(edges, framework.CreateEdge(secret, es, "ManagedBy"))
				}
			}
		}
	}
	return edges
}
