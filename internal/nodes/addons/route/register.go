package route

import (
	. "bloodhound-kube/internal/nodes/framework"

	routev1 "github.com/openshift/api/route/v1"
)

func Register() {
	RegisterResources(
		ResourceRegistration{
			GVK:          routev1.SchemeGroupVersion.WithKind("Route"),
			TypedBuilder: BuildRouteNode,
		},
	)
	RegisterDefaultCollections(
		CollectionTarget{
			Group:    routev1.GroupVersion.Group,
			Version:  routev1.GroupVersion.Version,
			Resource: "routes",
		},
	)
}
