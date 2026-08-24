package scc

import (
	. "bloodhound-kube/internal/nodes/framework"

	securityv1 "github.com/openshift/api/security/v1"
)

func Register() {
	RegisterResources(
		ResourceRegistration{
			GVK:          securityv1.SchemeGroupVersion.WithKind("SecurityContextConstraints"),
			TypedBuilder: BuildSecurityContextConstraintsNode,
		},
	)
}
