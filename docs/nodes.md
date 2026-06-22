# Nodes

Nodes represent entities in the Kubernetes cluster — pods, service accounts, secrets, and so on. Each node type maps one Kubernetes resource kind to a BloodHound graph node with stable IDs and typed properties.

## Node Inventory

Nodes are organized by domain. Each domain has its own subdirectory under `internal/nodes/<domain>/`.

### platform

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `BHK_Namespace` | `platform/namespace.go` | cluster | Metadata-only fetch (`FetchModeHintMetadata`) |
| `BHK_Node` | `platform/node.go` | cluster | Kubernetes worker/control-plane node; metadata-only fetch |
| `BHK_External` | `platform/external.go` | cluster | Synthetic node representing external traffic sources |

### rbac

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `BHK_ServiceAccount` | `rbac/serviceaccount.go` | namespace | |
| `BHK_User` | `rbac/user.go` | cluster | Created from RoleBinding/ClusterRoleBinding subjects |
| `BHK_Group` | `rbac/group.go` | cluster | Created from RoleBinding/ClusterRoleBinding subjects |
| `BHK_Role` | `rbac/role.go` | namespace | Namespaced RBAC role |
| `BHK_ClusterRole` | `rbac/clusterrole.go` | cluster | |
| `BHK_RoleBinding` | `rbac/rolebinding.go` | namespace | |
| `BHK_ClusterRoleBinding` | `rbac/clusterrolebinding.go` | cluster | |

### workload

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `BHK_Pod` | `workload/pod.go` | namespace | Full spec fetch |
| `BHK_Deployment` | `workload/deployment.go` | namespace | |
| `BHK_DaemonSet` | `workload/daemonset.go` | namespace | |
| `BHK_StatefulSet` | `workload/statefulset.go` | namespace | |
| `BHK_Job` | `workload/job.go` | namespace | |
| `BHK_CronJob` | `workload/cronjob.go` | namespace | |
| `BHK_Secret` | `workload/secret.go` | namespace | `data` omitted when `--redacted` |
| `BHK_ConfigMap` | `workload/configmap.go` | namespace | |

### networking

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `BHK_Service` | `networking/service.go` | namespace | |
| `BHK_Ingress` | `networking/ingress.go` | namespace | |
| `BHK_NetworkPolicy` | `networking/networkpolicy.go` | namespace | |
| `BHK_Gateway` | `networking/gateway.go` | namespace | Registered for both `gateway.networking.k8s.io/v1` and `v1beta1` |
| `BHK_HTTPRoute` | `networking/httproute.go` | namespace | Registered for `v1` and `v1beta1` |
| `BHK_GRPCRoute` | `networking/grpcroute.go` | namespace | Registered for `v1` and `v1alpha2` |
| `BHK_TCPRoute` | `networking/tcproute.go` | namespace | Registered for `v1alpha2` |
| `BHK_TLSRoute` | `networking/tlsroute.go` | namespace | Registered for `v1` and `v1alpha2` |

### mounts

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `BHK_PersistentVolume` | `mounts/pv.go` | cluster | Full spec fetch |
| `BHK_PersistentVolumeClaim` | `mounts/pvc.go` | namespace | Metadata-only fetch (`FetchModeHintMetadata`) |

### addons

| Kind | Go File | Scope | Notes |
|------|---------|-------|-------|
| `BHK_SecretStore` | `addons/external_secrets.go` | namespace | external-secrets operator |
| `BHK_ClusterSecretStore` | `addons/external_secrets.go` | cluster | external-secrets operator |
| `BHK_ExternalSecret` | `addons/external_secrets.go` | namespace | external-secrets operator |
| `BHK_SecurityContextConstraint` | `addons/security_context_constraints.go` | cluster | OpenShift only; builder implemented but not yet registered |

---

## Aggregate Nodes

Aggregate nodes are synthetic cluster-scoped nodes representing an entire resource class. They exist to avoid graph explosion when RBAC grants apply to all resources of a kind — for example, a role binding that grants `pods/exec` on all pods. Without aggregates, every single pod in the cluster would receive an inbound edge, making the graph unusable.

Aggregates are not collected from the API; they are always created by the parser at the start of the parse pass regardless of what was collected.

| Kind | Used By Edge |
|------|-------------|
| `BHK_AllPods` | `BHK_PodExec` (cluster-wide RBAC) |
| `BHK_AllSecrets` | `BHK_ReadSecret` (cluster-wide) |
| `BHK_AllConfigMaps` | `BHK_ReadConfigMap` (cluster-wide) |
| `BHK_AllServiceAccounts` | `BHK_Impersonate` (cluster-wide) |
| `BHK_AllNodes` | `BHK_WorkloadCreate` (cluster-wide) |
| `BHK_AllDeployments` | `BHK_WorkloadPatch` (cluster-wide) |
| `BHK_AllDaemonSets` | `BHK_WorkloadPatch` (cluster-wide) |
| `BHK_AllStatefulSets` | `BHK_WorkloadPatch` (cluster-wide) |
| `BHK_AllJobs` | `BHK_WorkloadPatch` (cluster-wide) |
| `BHK_AllCronJobs` | `BHK_WorkloadPatch` (cluster-wide) |
| `BHK_AllClusterRoles` | `BHK_RBACEscalate`, `BHK_RBACBind` (cluster-wide) |
| `BHK_AllUsers` | `BHK_Impersonate` (cluster-wide user impersonation) |
| `BHK_AllGroups` | `BHK_Impersonate` (cluster-wide group impersonation) |
| `BHK_AllRoles` | `BHK_RBACEscalate`, `BHK_RBACBind` (namespace-scoped) |

Aggregate builders live in `internal/nodes/platform/aggregates.go`. They use `CoreEntry{Cluster: true}` so edge rules can look them up from the cluster-scoped index. Namespace-scoped aggregates (e.g. `BHK_AllPods` within a specific namespace) are also created per-namespace and stored in the namespace index.

---

## Secondary Types (for Cypher Queries)

Some node kinds carry additional secondary type labels. These are not standalone node kinds — they are extra labels on existing nodes that enable broader Cypher queries across related types.

| Secondary Kind | Applied To | Purpose |
|----------------|-----------|---------|
| `BHK_Identity` | `BHK_ServiceAccount`, `BHK_User`, `BHK_Group` | Abstract principal type shared by all identity kinds. Use `MATCH (n:BHK_Identity)` to query across all principals regardless of whether they are a ServiceAccount, User, or Group. |
| `BHK_Aggregate` | All `BHK_All*` nodes | Shared label on all aggregate nodes. Use `MATCH (n:BHK_Aggregate)` to query all aggregate targets in a single Cypher pattern. |

These secondary types are defined in `config/schema.json` with `is_display_kind: false` (they do not appear as standalone node types in the BloodHound UI).

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

- `ListenerSet` / `ListenerSetGroup` (Gateway API extension)
- OpenShift `BHK_SecurityContextConstraint` node (builder exists in `addons/security_context_constraints.go` but is not yet registered)
- Istio resources (stub file exists but is not implemented)
