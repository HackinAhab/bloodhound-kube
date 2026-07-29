package rbac

import (
	"bloodhound-kube/internal/model"
	"bloodhound-kube/internal/nodes/platform"
	"bloodhound-kube/internal/nodes/workload"
)

var rbacReadSecretsEdgesRule = simpleAccessRule[workload.Secret, platform.AllSecrets]{
	name:         "rbac_read_secrets",
	resourceKeys: []string{"secrets"},
	verbs:        []string{"get", "list", "watch"},
	edgeKind:     "BHK_ReadSecret",
	props: map[string]any{
		"Description": "Identity has RBAC permissions to read secrets.",
	},
	namespacedTargets: func(space *model.Namespace) []workload.Secret { return space.Secrets },
	namespacedAll:     func(space *model.Namespace) []platform.AllSecrets { return space.AllSecrets },
	clusterAll:        func(c *model.Cluster) []platform.AllSecrets { return c.AllSecrets },
}

var rbacReadConfigMapsEdgesRule = simpleAccessRule[workload.ConfigMap, platform.AllConfigMaps]{
	name:         "rbac_read_configmaps",
	resourceKeys: []string{"configmaps"},
	verbs:        []string{"get", "list", "watch"},
	edgeKind:     "BHK_ReadConfigMap",
	props: map[string]any{
		"Description": "Identity has RBAC permissions to read configmaps.",
	},
	namespacedTargets: func(space *model.Namespace) []workload.ConfigMap { return space.ConfigMaps },
	namespacedAll:     func(space *model.Namespace) []platform.AllConfigMaps { return space.AllConfigMaps },
	clusterAll:        func(c *model.Cluster) []platform.AllConfigMaps { return c.AllConfigMaps },
}
