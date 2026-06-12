package workload

import "bloodhound-kube/internal/edges/framework"

func Register(reg *framework.Registry) {
	reg.Register(podEdgesRule{})
	reg.Register(configMapEdgesRule{})
	reg.Register(secretEdgesRule{})
	reg.Register(deploymentEdgesRule{})
	reg.Register(daemonSetEdgesRule{})
	reg.Register(statefulSetEdgesRule{})
	reg.Register(jobEdgesRule{})
	reg.Register(cronJobEdgesRule{})
}
