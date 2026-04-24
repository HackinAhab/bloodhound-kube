# Relationship definitions
I found KubeHound's relationship naming scheme to be intuitive and have adopted it where it made sense. I have also included links to the source used as a reference for each relationship policy in the individual policy files. 

## General Relationships for Contextual Awareness
- ScheduledOn: Pod -> Node
- Managedby: Pod -> Controller (e.g. Deployment, StatefulSet, DaemonSet)
- AppliesTo: NetworkPolicy -> Pod
- RoutesTo: Ingress|HTTPRoute|GRPCRoute|TCPRoute|TLSRoute -> Service
- RoutesTo: Gateway -> HTTPRoute|GRPCRoute|TCPRoute|TLSRoute
- ExternalRoutesTo: External -> Service|Ingress|Gateway
- MountedBy: Volume -> Pod
- EnvVars: Secret|ConfigMap -> Pod # TODO: Implement ConfigMap -> Pod
- BoundTo:
- RoleBound: Role/ClusterRole -> ServiceAccount
- SAToken

## Container Escapes
- CE_NSENTER: Pod -> Node
    - 
- CE_PRIV_MOUNT: Pod -> Node
- CE_SYS_PTRACE: Pod -> Node
- CE_UMH_CORE_PATTERN: Pod -> Node

## Lateral Movement
- LM_HOST_MOUNT_KUBELET: Pod -> Node -> Node

## RBAC
- SAImpersonate
- WorkloadCreate
- RBACCreate
- PodExec
- PodDebug
- WorkloadPatch
- NodeProxy
- SAReadSecret
- SAReadConfigMap

## Adding new relationship rules
Add rules under a domain package in `internal/edges/rules/*` and register them explicitly from that package's `Register` function. The top-level `BuildEdges` wiring in `internal/edges/edge_registry.go` calls each domain register function.

Example domain registration:

```go
package mydomain

import "bloodhound-kube/internal/edges/framework"

func Register(reg *framework.Registry) {
	reg.Register(exampleRule{})
}

```

Example rule implementation:

```go
package mydomain

import "bloodhound-kube/internal/model"
import "bloodhound-kube/internal/edges/framework"

type exampleRule struct{}

func (r exampleRule) Name() string { return "example" }

func (r exampleRule) Apply(ctx *framework.Context) []model.BloodHoundEdge {
	if ctx == nil || ctx.Core == nil {
		return nil
	}
	// Build edges with framework.CreateEdge/framework.CreateEdgeWithProperties
	return nil
}
```
