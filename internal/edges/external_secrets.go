package edges

import (
	"strings"

	"bloodhound-kube/internal/model"
)

type externalSecretsEdgesRule struct{}

func (r externalSecretsEdgesRule) Name() string {
	return "external_secrets"
}

func init() {
	RegisterEdgeRule(externalSecretsEdgesRule{})
}

func (r externalSecretsEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
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
			storeName := es.StoreName
			if storeName != "" {
				storeKind := strings.ToLower(es.StoreKind)
				if storeKind == "" {
					storeKind = "secretstore"
				}
				if storeKind == "secretstore" {
					if store := secretStores[storeName]; store != nil {
						edges = append(edges, CreateEdge(es, store, "ManagedBy"))
					}
				}
				if storeKind == "clustersecretstore" {
					if store := ctx.Index.ClusterSecretStoresByName[storeName]; store != nil {
						edges = append(edges, CreateEdge(es, store, "ManagedBy"))
					}
				}
			}

			for j := range space.Secrets {
				secret := &space.Secrets[j]
				targetName := es.TargetName
				if targetName != "" {
					if secret.Name == targetName {
						edges = append(edges, CreateEdge(secret, es, "ManagedBy"))
					}
					continue
				}
				if secret.Name == es.Name {
					edges = append(edges, CreateEdge(secret, es, "ManagedBy"))
				}
			}
		}
	}
	return edges
}
