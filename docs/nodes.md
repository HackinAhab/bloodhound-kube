# Nodes

Nodes represent entities in the Kubernetes cluster — pods, service accounts, secrets, and so on. Each node type maps one Kubernetes resource kind to a BloodHound graph node with stable IDs and typed properties.

## Node Inventory

Nodes are organized by domain. Each domain has its own subdirectory under `internal/nodes/<domain>/`.

### platform

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `Namespace` | `platform/namespace.go` | cluster | Metadata-only fetch (`FetchModeHintMetadata`) |
| `Node` | `platform/node.go` | cluster | Kubernetes worker/control-plane node; metadata-only fetch |
| `External` | `platform/external.go` | cluster | Synthetic node representing external traffic sources |

### rbac

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `ServiceAccount` | `rbac/serviceaccount.go` | namespace | |
| `Role` | `rbac/role.go` | namespace | Namespaced RBAC role |
| `ClusterRole` | `rbac/clusterrole.go` | cluster | |
| `RoleBinding` | `rbac/rolebinding.go` | namespace | |
| `ClusterRoleBinding` | `rbac/clusterrolebinding.go` | cluster | |

### workload

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `Pod` | `workload/pod.go` | namespace | Full spec fetch |
| `Deployment` | `workload/deployment.go` | namespace | |
| `DaemonSet` | `workload/daemonset.go` | namespace | |
| `StatefulSet` | `workload/statefulset.go` | namespace | |
| `Job` | `workload/job.go` | namespace | |
| `CronJob` | `workload/cronjob.go` | namespace | |
| `Secret` | `workload/secret.go` | namespace | `data` omitted when `--redacted` |
| `ConfigMap` | `workload/configmap.go` | namespace | |

### networking

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `Service` | `networking/service.go` | namespace | |
| `Ingress` | `networking/ingress.go` | namespace | |
| `NetworkPolicy` | `networking/networkpolicy.go` | namespace | |
| `Gateway` | `networking/gateway.go` | namespace | Registered for both `gateway.networking.k8s.io/v1` and `v1beta1` |
| `HTTPRoute` | `networking/httproute.go` | namespace | Registered for `v1` and `v1beta1` |
| `GRPCRoute` | `networking/grpcroute.go` | namespace | Registered for `v1` and `v1alpha2` |
| `TCPRoute` | `networking/tcproute.go` | namespace | Registered for `v1alpha2` |
| `TLSRoute` | `networking/tlsroute.go` | namespace | Registered for `v1` and `v1alpha2` |

### mounts

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `PersistentVolume` | `mounts/pv.go` | cluster | Full spec fetch |
| `PersistentVolumeClaim` | `mounts/pvc.go` | namespace | Metadata-only fetch (`FetchModeHintMetadata`) |

### addons

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `SecretStore` | `addons/external_secrets.go` | namespace | external-secrets operator |
| `ClusterSecretStore` | `addons/external_secrets.go` | cluster | external-secrets operator |
| `ExternalSecret` | `addons/external_secrets.go` | namespace | external-secrets operator |
| `SecurityContextConstraints` | `addons/security_context_constraints.go` | cluster | OpenShift only; builder implemented but not yet registered |

---

## Aggregate Nodes

Aggregate nodes are synthetic cluster-scoped nodes representing an entire resource class. They exist to avoid graph explosion when RBAC grants apply to all resources of a kind — for example, a role binding that grants `pods/exec` on all pods. Without aggregates, every single pod in the cluster would receive an inbound edge, making the graph unusable.

Aggregates are not collected from the API; they are always created by the parser at the start of the parse pass regardless of what was collected.

| Kind | Used By Edge |
|------|-------------|
| `AllPods` | `PodExec`, `PodDebug` (cluster-wide RBAC) |
| `AllSecrets` | `SAReadSecret` (cluster-wide) |
| `AllConfigMaps` | `ReadConfigMap` (cluster-wide) |
| `AllServiceAccounts` | `ImpersonateSA` (cluster-wide) |
| `AllNodes` | `WorkloadCreate` (cluster-wide) |
| `AllDeployments` | `WorkloadPatch` (cluster-wide) |
| `AllDaemonSets` | `WorkloadPatch` (cluster-wide) |
| `AllStatefulSets` | `WorkloadPatch` (cluster-wide) |
| `AllJobs` | `WorkloadPatch` (cluster-wide) |
| `AllCronJobs` | `WorkloadPatch` (cluster-wide) |

Aggregate builders live in `internal/nodes/platform/aggregates.go`. They use `CoreEntry{Cluster: true}` so edge rules can look them up from the cluster-scoped index.

---

## CoreEntry and CoreFacts

Node builders return a `BuildResult` containing two things:
- `Node NodeResult` — the graph node (ID, kinds, properties)
- `Core []CoreEntry` — typed structs that edge rules look up during the edge pass

```go
type BuildResult struct {
    Node NodeResult
    Core []CoreEntry
}

type CoreEntry struct {
    Namespace string // set when Cluster is false
    Cluster   bool   // true → cluster-scoped index; false → namespace index
    Data      any    // a typed struct from internal/nodes/<domain>/
}
```

All `CoreEntry` items from a parse pass are accumulated into `model.CoreFacts`:

```go
// Namespace-scoped resources (CoreEntry.Cluster == false)
type Namespace struct {
    Pods             []workload.Pod
    ServiceAccounts  []rbac.ServiceAccount
    Secrets          []workload.Secret
    ConfigMaps       []workload.ConfigMap
    Services         []networking.Service
    // ... (see internal/model/coretypes.go for the full list)
}

// Cluster-scoped resources (CoreEntry.Cluster == true)
type Cluster struct {
    Nodes                      []platform.Node
    ClusterRoles               []rbac.ClusterRole
    ClusterRoleBindings        []rbac.ClusterRoleBinding
    PersistentVolumes          []mounts.PersistentVolume
    SecurityContextConstraints []addons.SecurityContextConstraints
    // ... aggregate nodes (AllPods, AllSecrets, ...)
}
```

Dispatch into the correct field is handled by reflection in `internal/model/corefacts_adders.go`. If you add a new typed struct that edge rules need to look up, add the slice field to `model.Namespace` or `model.Cluster` and register an adder in `corefacts_adders.go`.

Edge rules receive `CoreFacts` via `*framework.Context` and can also use the pre-built `*model.CoreIndex` (pointer maps keyed by name/namespace for O(1) lookup).

---

## How to Add a Node Type

### 1. Create the builder file

```
internal/nodes/<domain>/my_kind.go
```

Define a typed struct for the core data and a builder function:

```go
package <domain>

import (
    . "bloodhound-kube/internal/nodes/framework"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/runtime"
)

// Typed struct — fields used by edge rules
type MyKind struct {
    GraphNodeBase
    MyProperty string
}

// TypedBuilder — preferred when a typed k8s API object is available
func BuildMyKindNode(obj runtime.Object) (BuildResult, bool) {
    res, ok := obj.(*corev1.MyKind)
    if !ok || res == nil {
        return BuildResult{}, false
    }
    name := res.Name
    if name == "" {
        return BuildResult{}, false
    }
    namespace := res.Namespace
    labelsMap := StringMapToAnyMap(res.Labels)
    annotationsMap := StringMapToAnyMap(res.Annotations)

    myProp := string(res.Spec.SomeField)

    properties := map[string]any{
        "name":        name,
        "namespace":   namespace,
        "labels":      MapToSortedList(labelsMap),
        "annotations": MapToSortedList(annotationsMap),
        "myProperty":  myProp,
    }

    base := NewGraphNodeBase("MyKind", namespace, name, labelsMap, annotationsMap)

    core := CoreEntry{
        Namespace: namespace,
        Cluster:   false, // true for cluster-scoped resources
        Data: MyKind{
            GraphNodeBase: base,
            MyProperty:    myProp,
        },
    }

    return BuildResult{
        Node: NewNodeResult(base, properties),
        Core: []CoreEntry{core},
    }, true
}
```

For CRDs or resources without a typed Go struct, use the map-based `Builder` signature instead (`func(resource map[string]any) (BuildResult, bool)`).

### 2. Register the builder

In your domain's `register.go`, add a line inside `Register(reg *framework.Registry)`:

```go
// For typed k8s objects (preferred):
reg.RegisterTyped(corev1.SchemeGroupVersion.WithKind("MyKind"), BuildMyKindNode)

// For metadata-only collection (saves bandwidth):
reg.RegisterTypedWithFetchMode(gvk, BuildMyKindNode, framework.FetchModeHintMetadata)

// For map-based builders (CRDs without typed structs):
reg.RegisterTypedFromMap(gvk, BuildMyKindNode)

// For kind-name dispatch only (no GVK, untyped):
reg.Register("MyKind", BuildMyKindNode)
```

### 3. Wire CoreFacts if edge rules need it

If edge rules need to look up this node type:

1. Add the slice to `internal/model/coretypes.go`:

```go
type Namespace struct {
    // ...
    MyKinds []<domain>.MyKind
}
```

2. Register an adder in `internal/model/corefacts_adders.go`:

```go
registerNamespacedFactAdder(func(c *CoreFacts, v <domain>.MyKind) {
    ns := c.ensureNamespace(v.Namespace)
    ns.MyKinds = append(ns.MyKinds, v)
})
```

Use `registerClusterFactAdder` for cluster-scoped types.

### 4. Verify domain wiring

Confirm `internal/nodes/node_registry.go` imports your domain package (each domain's `Register` is called from `ensureRegistered()`).

---

## Object Types Not Covered

The following are not currently implemented:

- Non-ServiceAccount RBAC subjects (User, Group)
- `ListenerSet` / `ListenerSetGroup` (Gateway API extension)
- OpenShift `SecurityContextConstraints` node (builder exists in `addons/security_context_constraints.go` but is not yet registered)
- Istio resources (stub file exists but is not implemented)
