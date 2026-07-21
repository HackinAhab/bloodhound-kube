package rbac

import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/rbac"
)

var rbacSATokenRequestEdgesRule = simpleAccessRule[rbac.ServiceAccount, platform.AllServiceAccounts]{
	name:         "rbac_sa_token_request",
	resourceKeys: []string{"serviceaccounts/token"},
	verbs:        []string{"create"},
	edgeKind:     "BHK_SATokenRequest",
	props: map[string]any{
		"Description": "Identity has RBAC permission to create ServiceAccount tokens (TokenRequest), allowing it to mint API tokens for any ServiceAccount in scope.",
		"Reference":   "https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#bound-service-account-tokens",
	},
	namespacedTargets: func(space *model.Namespace) []rbac.ServiceAccount { return space.ServiceAccounts },
	namespacedAll:     func(space *model.Namespace) []platform.AllServiceAccounts { return space.AllServiceAccounts },
	clusterAll:        func(c *model.Cluster) []platform.AllServiceAccounts { return c.AllServiceAccounts },
}
