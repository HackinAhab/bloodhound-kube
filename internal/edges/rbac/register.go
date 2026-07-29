package rbac

import "bloodhound-kube/internal/edges/framework"

func Register(reg *framework.Registry) {
	reg.Register(rbacEdgesRule{})
	reg.Register(rbacReadSecretsEdgesRule)
	reg.Register(rbacReadConfigMapsEdgesRule)
	reg.Register(rbacImpersonateEdgesRule{})
	reg.Register(rbacPodExecEdgesRule)
	reg.Register(rbacPodDebugEdgesRule)
	reg.Register(rbacReadLogsEdgesRule)
	reg.Register(rbacPatchWorkloadEdgesRule{})
	reg.Register(rbacCreateEdgesRule{})
	reg.Register(rbacCreateWorkloadEdgesRule{})
	reg.Register(rbacNodeProxyEdgesRule{})
	reg.Register(rbacPodPortForwardEdgesRule)
	reg.Register(rbacPodAttachEdgesRule)
	reg.Register(rbacSATokenRequestEdgesRule)
	reg.Register(rbacEscalateBindEdgesRule{})
	reg.Register(rbacSCCUsageEdgesRule{})
}
