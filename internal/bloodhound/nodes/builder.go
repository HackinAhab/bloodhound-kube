package nodes

import (
	"bloodhound-kube/internal/bloodhound"
)

func RegisterParsers() {
	bloodhound.DefaultRegistry.Register(NewNodeParser())
	bloodhound.DefaultRegistry.Register(NewSecretParser())
	bloodhound.DefaultRegistry.Register(NewServiceParser())
	bloodhound.DefaultRegistry.Register(NewConfigMapParser())
	bloodhound.DefaultRegistry.Register(NewIngressParser())
	bloodhound.DefaultRegistry.Register(NewGatewayParser())
	bloodhound.DefaultRegistry.Register(NewNetworkPolicyParser())
	bloodhound.DefaultRegistry.Register(NewRBACParser())
	bloodhound.DefaultRegistry.Register(NewCRDParser())
}

func GetAllSupportedKinds() []string {
	parsers := bloodhound.DefaultRegistry.GetAllParsers()
	var kinds []string

	for _, parser := range parsers {
		kinds = append(kinds, parser.GetSupportedKinds()...)
	}

	return kinds
}
