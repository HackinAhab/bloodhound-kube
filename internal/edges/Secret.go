package edges

import "bloodhound-kube/internal/model"

type secretEdgesRule struct{}

func (r secretEdgesRule) Name() string {
	return "secret"
}

func (r secretEdgesRule) Apply(ctx *EdgeContext) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	var edges []model.BloodHoundEdge
	for ns, space := range ctx.Core.Namespaces {
		if space == nil {
			continue
		}
		serviceAccounts := ctx.Index.ServiceAccountsByNamespace[ns]
		for i := range space.Secrets {
			secret := &space.Secrets[i]
			if secret.SecretType != "kubernetes.io/service-account-token" {
				continue
			}
			for _, sa := range serviceAccounts {
				for _, secretName := range sa.Secrets {
					if secretName == secret.Name {
						edges = append(edges, CreateEdge(secret, sa, "LongLivedToken"))
					}
				}
			}
		}
	}
	return edges
}

func init() {
	RegisterEdgeRule(secretEdgesRule{})
}
