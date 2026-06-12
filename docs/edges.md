# Edge Rules Reference

Each rule lives in `internal/edges/<domain>/` and implements the `Rule` interface. This document describes every rule: what misconfiguration or attack path it models, the conditions that trigger it, and the edge type emitted between which nodes.

Edge naming follows [KubeHound's](https://kubehound.io/reference/attacks/) convention where applicable.

---

## RBAC (`internal/edges/rbac/`)

### `rbac_base` — Role Bindings
**File:** `base_bindings.go`

Maps the structural RBAC graph. For every RoleBinding/ClusterRoleBinding that binds a Role or ClusterRole to a ServiceAccount, a `RoleBound` edge is created.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `RoleBound` | ServiceAccount → Role/ClusterRole | A RoleBinding or ClusterRoleBinding links the ServiceAccount subject to the role |

Each `RoleBound` edge carries `bindingKind`, `bindingName`, `bindingNamespace`, `roleKind`, and `roleName` properties identifying the binding that produced it (taken from the first contributing binding after a deterministic sort by `(namespace, name, kind)`). When multiple bindings link the same role and ServiceAccount, they are merged into a single edge to keep graph cardinality bounded; the full list is exposed as a structured `bindings` array (`[{kind, name, namespace, roleKind, roleName}, ...]`) and a `bindingCount` integer.

---

### `rbac_impersonate` — Service Account Impersonation
**File:** `impersonate.go`  
**Reference:** https://kubehound.io/reference/attacks/IDENTITY_IMPERSONATE/

A ServiceAccount with `impersonate` on the `serviceaccounts` resource can act as any other ServiceAccount, bypassing its own privilege constraints. The same rule also detects `impersonate` on `users` and `groups` resources, which allows impersonating arbitrary user identities or groups (including `system:masters`). Cluster-scoped bindings emit edges to the `AllServiceAccounts` aggregate node.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `ImpersonateSA` | ServiceAccount → ServiceAccount | SA has `impersonate` verb on `serviceaccounts` resource via a RoleBinding or ClusterRoleBinding |
| `ImpersonateSA` | ServiceAccount → AllServiceAccounts | Cluster-scoped binding with wildcard ServiceAccount access |
| `ImpersonateUsers` | ServiceAccount → AllServiceAccounts | SA has `impersonate` on `users` resource via any RoleBinding or ClusterRoleBinding |
| `ImpersonateGroups` | ServiceAccount → AllServiceAccounts | SA has `impersonate` on `groups` resource via any RoleBinding or ClusterRoleBinding |

---

### `rbac_pod_exec` — Pod Exec
**File:** `pod_access.go`  
**Reference:** https://kubehound.io/reference/attacks/POD_EXEC/

`create` on `pods/exec` allows running arbitrary commands in a pod's containers via `kubectl exec`. Cluster-scoped bindings emit an edge to the `AllPods` aggregate.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `PodExec` | ServiceAccount → Pod | SA has `create` on `pods/exec` |
| `PodExec` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod access |

---

### `rbac_pod_debug` — Pod Debug (Ephemeral Containers)
**File:** `pod_access.go`  
**Reference:** https://kubehound.io/reference/attacks/POD_ATTACH/

`update` on `pods/ephemeralcontainers` allows attaching an ephemeral debug container to a running pod (this is the permission `kubectl debug` requires), which can be used to inspect or exfiltrate data.

> **Note:** Unlike `PodExec`, `PodAttach`, and `PodPortForward`, this rule does not emit a cluster-scoped `AllPods` aggregate edge. Only individual pod targets are emitted.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `PodDebug` | ServiceAccount → Pod | SA has `update` on `pods/ephemeralcontainers` |

---

### `rbac_read_logs` — Pod Logs Read
**File:** `pod_logs.go`

`get`, `list`, or `watch` on `pods/log` allows reading container stdout/stderr from any pod in scope. Logs frequently leak credentials, tokens, request payloads, and internal endpoints, so broad log access is effectively a cluster-wide credential-harvesting primitive. Cluster-scoped bindings emit an edge to the `AllPods` aggregate.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `ReadLogs` | ServiceAccount → Pod | SA has `get`/`list`/`watch` on `pods/log` |
| `ReadLogs` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod log access |

---

### `rbac_read_secrets` — Secret Read
**File:** `read.go`

`get`, `list`, or `watch` on `secrets` allows reading Kubernetes Secret values, which may contain credentials, API keys, TLS private keys, or other sensitive material. Cluster-scoped bindings emit an edge to `AllSecrets`.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `SAReadSecret` | ServiceAccount → Secret | SA has `get`/`list`/`watch` on `secrets` |
| `SAReadSecret` | ServiceAccount → AllSecrets | Cluster-scoped, wildcard secret access |

---

### `rbac_read_configmaps` — ConfigMap Read
**File:** `read.go`

`get`, `list`, or `watch` on `configmaps` allows reading ConfigMap data, which sometimes contains sensitive configuration, internal endpoints, or bootstrap credentials. Cluster-scoped bindings emit an edge to `AllConfigMaps`.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `ReadConfigMap` | ServiceAccount → ConfigMap | SA has `get`/`list`/`watch` on `configmaps` |
| `ReadConfigMap` | ServiceAccount → AllConfigMaps | Cluster-scoped, wildcard configmap access |

---

### `rbac_create` — RBAC Binding Creation
**File:** `workload_create.go`

`create` on `rolebindings` or `clusterrolebindings` allows a ServiceAccount to grant itself or other identities arbitrary roles, enabling privilege escalation.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `RBACCreate` | ServiceAccount → Role/ClusterRole | SA has `create` on `rolebindings` (namespaced) or `clusterrolebindings` (cluster-scoped) |

---

### `rbac_create_workload` — Workload Creation
**File:** `workload_create.go`  
**Reference:** https://kubernetes.io/docs/reference/access-authn-authz/rbac/#referring-to-resources

`create` on workload resources (pods, deployments, daemonsets, statefulsets, jobs, cronjobs) allows scheduling privileged pods on any node, which is equivalent to node compromise.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `WorkloadCreate` | ServiceAccount → Node | SA has `create` on pods, deployments, daemonsets, statefulsets, jobs, or cronjobs |

---

### `rbac_patch_workload` — Workload Patch/Update
**File:** `workload_patch.go`  
**Reference:** https://kubehound.io/reference/attacks/POD_PATCH/

`patch` or `update` on running workloads allows modifying a pod spec (e.g. injecting a new container image, adding volume mounts, changing the command). Cluster-scoped bindings emit edges to aggregate nodes.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `WorkloadPatch` | ServiceAccount → Pod | SA has `patch`/`update` on `pods` |
| `WorkloadPatch` | ServiceAccount → Deployment | SA has `patch`/`update` on `deployments` |
| `WorkloadPatch` | ServiceAccount → DaemonSet | SA has `patch`/`update` on `daemonsets` |
| `WorkloadPatch` | ServiceAccount → StatefulSet | SA has `patch`/`update` on `statefulsets` |
| `WorkloadPatch` | ServiceAccount → Job | SA has `patch`/`update` on `jobs` |
| `WorkloadPatch` | ServiceAccount → CronJob | SA has `patch`/`update` on `cronjobs` |
| `WorkloadPatch` | ServiceAccount → AllPods | Cluster-scoped binding with wildcard pod patch access |
| `WorkloadPatch` | ServiceAccount → AllDeployments | Cluster-scoped binding with wildcard deployment patch access |
| `WorkloadPatch` | ServiceAccount → AllDaemonSets | Cluster-scoped binding with wildcard daemonset patch access |
| `WorkloadPatch` | ServiceAccount → AllStatefulSets | Cluster-scoped binding with wildcard statefulset patch access |
| `WorkloadPatch` | ServiceAccount → AllJobs | Cluster-scoped binding with wildcard job patch access |
| `WorkloadPatch` | ServiceAccount → AllCronJobs | Cluster-scoped binding with wildcard cronjob patch access |

---

### `rbac_node_proxy` — Node Proxy / RCE via Kubelet API
**File:** `node_proxy.go`  
**Reference:** https://grahamhelton.com/blog/nodes-proxy-rce

`get` on `nodes/proxy` allows proxying HTTP requests directly to the kubelet API on any node. This can be used to list and exec into pods, retrieve secrets from the kubelet, or interact with the container runtime. Cluster-scoped bindings (which also check `create` and `proxy` verbs) are flagged as `NodeProxyRCE` to distinguish higher-severity access.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `NodeProxy` | ServiceAccount → Pod | SA has `get` on `nodes/proxy` via a namespaced binding; pod is on a matching node |
| `NodeProxyRCE` | ServiceAccount → Pod | SA has `get`/`create`/`proxy` on `nodes/proxy` via a cluster-scoped binding |
| `NodeProxyRCE` | ServiceAccount → AllNodes | Cluster-scoped binding with wildcard node access (`all` flag set) |

---

### `rbac_pod_portforward` — Pod Port Forward
**File:** `pod_access.go`

`create` on `pods/portforward` allows forwarding a local port to a port inside a pod, giving direct TCP access to services running in the container — useful for lateral movement to internal services. Cluster-scoped bindings emit an edge to the `AllPods` aggregate.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `PodPortForward` | ServiceAccount → Pod | SA has `create` on `pods/portforward` |
| `PodPortForward` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod access |

---

### `rbac_pod_attach` — Pod Attach
**File:** `pod_access.go`

`create` on `pods/attach` allows attaching to the stdin/stdout of a running container, giving interactive shell access equivalent to `kubectl attach`. Cluster-scoped bindings emit an edge to the `AllPods` aggregate.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `PodAttach` | ServiceAccount → Pod | SA has `create` on `pods/attach` |
| `PodAttach` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod access |

---

### `rbac_sa_token_request` — ServiceAccount Token Request
**File:** `sa_token_request.go`  
**Reference:** https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#bound-service-account-tokens

`create` on `serviceaccounts/token` allows minting API tokens for any ServiceAccount in scope via the TokenRequest API. An attacker can use this to assume the identity of a higher-privileged ServiceAccount without needing its existing credentials. Cluster-scoped bindings emit an edge to `AllServiceAccounts`.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `SATokenRequest` | ServiceAccount → ServiceAccount | SA has `create` on `serviceaccounts/token` via a RoleBinding |
| `SATokenRequest` | ServiceAccount → AllServiceAccounts | Cluster-scoped, wildcard serviceaccount access |

---

### `rbac_escalate_bind` — Role Escalation and Binding
**File:** `escalate_bind.go`  
**Reference:** https://kubernetes.io/docs/reference/access-authn-authz/rbac/#privilege-escalation-prevention-and-bootstrapping

`escalate` allows modifying a role to include permissions the caller does not currently hold, bypassing Kubernetes' escalation prevention. `bind` allows creating a RoleBinding for any role without holding that role's permissions. Both verbs on `roles`/`clusterroles` are privilege-escalation primitives.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `RBACEscalate` | ServiceAccount → Role/ClusterRole | SA has `escalate` on `roles` or `clusterroles` (namespaced binding, or cluster-scoped binding with named targets) |
| `RBACBind` | ServiceAccount → Role/ClusterRole | SA has `bind` on `roles` or `clusterroles` without `escalate` (namespaced binding, or cluster-scoped binding with named targets) |
| `RBACEscalate` | ServiceAccount → AllClusterRoles | Cluster-scoped binding with wildcard ClusterRole access (`escalate`) |
| `RBACBind` | ServiceAccount → AllClusterRoles | Cluster-scoped binding with wildcard ClusterRole access (`bind` only) |
| `RBACEscalate` | ServiceAccount → AllRoles | Cluster-scoped binding with wildcard Role access (`escalate`) |
| `RBACBind` | ServiceAccount → AllRoles | Cluster-scoped binding with wildcard Role access (`bind` only) |

---

### `rbac_scc_usage` — SecurityContextConstraints Usage
**File:** `scc_usage.go`  
**Reference:** https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html

OpenShift-specific. `use` on `securitycontextconstraints` (in the `security.openshift.io` API group) allows a ServiceAccount to request admission under that SCC. Combined with `EnforcedSCC` (SCC → Pod), this reveals the full privilege chain: SA → SCC → Pod. Only ClusterRoleBindings are evaluated because SCCs are cluster-scoped resources.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `SCCUse` | ServiceAccount → SecurityContextConstraints | SA has `use` on `securitycontextconstraints` via a ClusterRoleBinding |

---

## Security (`internal/edges/security/`)

### `capabilities` — Dangerous Linux Capabilities
**File:** `capabilities_escapes.go`

Checks each pod's containers for dangerous capabilities explicitly added via `securityContext.capabilities.add`. Each capability is emitted as its own edge type.

| Edge | Source → Target | Trigger | Impact |
|------|----------------|---------|--------|
| `CAP_SYS_ADMIN` | Pod → Node | Container has `CAP_SYS_ADMIN` added | Broad privilege escalation; can mount filesystems, load eBPF programs, bypass namespacing |
| `CAP_NET_ADMIN` | Pod → Node | Container has `CAP_NET_ADMIN` added | Network administration; can intercept traffic, modify interfaces and routes |
| `CAP_SYS_MODULE` | Pod → Node | Container has `CAP_SYS_MODULE` added | Load/unload kernel modules; effectively full kernel access |
| `CAP_SYS_PTRACE` | Pod → Node | Container has `CAP_SYS_PTRACE` added | Trace and debug other processes; enables code injection and secret theft from host processes |
| `CAP_SYS_RAWIO` | Pod → Node | Container has `CAP_SYS_RAWIO` added | Raw I/O; can access and modify raw disk data and memory |

---

### `container_escapes` — Container Escape Paths
**File:** `capabilities_escapes.go`

Checks pods for configurations that enable escaping the container boundary to the host node.

| Edge | Source → Target | Trigger | Attack |
|------|----------------|---------|--------|
| `CE_PRIV_MOUNT` | Pod → Node | Any container has `privileged: true` | Privileged containers can mount the host filesystem and interact directly with host devices |
| `CE_NSENTER` | Pod → Node | Pod has `hostPID: true` AND a privileged container | An attacker can use `nsenter` to switch into the host PID namespace and execute commands as root on the node |
| `CE_SYS_PTRACE` | Pod → Node | Privileged container, OR (hostPID + CAP_SYS_PTRACE + CAP_SYS_ADMIN) | Attach to a host process via ptrace to inject shellcode or steal credentials |
| `CE_UMH_CORE_PATTERN` | Pod → Node | HostPath volume at `/proc`, `/proc/sys`, or `/proc/sys/kernel` with a writable container mount | Write to `/proc/sys/kernel/core_pattern` to execute arbitrary commands on the host when any process crashes |
| `MOUNT_CONTAINER_SOCKET` | Pod → Node | HostPath volume path ends in `.sock` (e.g. `/run/containerd/containerd.sock`) | Access the container runtime socket to create privileged containers or escape to the host |
| `CE_VAR_LOG_SYMLINK` | Pod → Node | HostPath at `/var/log` or `/var`, container is privileged and not `runAsNonRoot` | Create symlinks in the mounted log directory pointing to sensitive host files, then read them via the Kubernetes log API |
| `CE_HOST_IPC` | Pod → Node | Pod has `hostIPC: true` AND (privileged container OR `CAP_SYS_ADMIN`) | Shares host IPC namespace; access to host shared memory, semaphores, and message queues enables IPC-based process injection |
| `CE_HOST_NETWORK` | Pod → Node | Pod has `hostNetwork: true` | Container shares host network namespace; can bind host ports, intercept node-level traffic, and reach node-local services |
| `CE_SHARE_PROC_NS` | Pod → Node | Pod has `shareProcessNamespace: true` AND (privileged container OR `CAP_SYS_PTRACE`) | All containers share a PID namespace; a privileged container can ptrace other containers to inject code or steal secrets from memory |

---

### `security_context_constraints` — OpenShift SCC Enforcement
**File:** `scc_hostports.go`

OpenShift-specific. Links a SecurityContextConstraints object to every pod that was admitted under it (indicated by the `openshift.io/scc` annotation). Used for contextual awareness of what security posture was enforced.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `EnforcedSCC` | SecurityContextConstraints → Pod | Pod has `openshift.io/scc` annotation matching an SCC name |

---

### `host_ports` — Container Host Port Exposure
**File:** `scc_hostports.go`

Containers with `hostPort` set bind directly to a port on the node's network interface, bypassing Service and NetworkPolicy controls. This exposes the container directly to any network reachable to the node.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `HostPort` | Node → Pod | Any container in the pod has `hostPort > 0` |
| `ExternalHostPort` | External → Node | Same — annotates that the node itself becomes externally reachable on that port |

---

## Mounts (`internal/edges/mounts/`)

### `lateral_movement_host_mount_read` — Sensitive Host Path Read
**File:** `host_mount.go`  
**Reference:** https://kubehound.io/reference/attacks/EXPLOIT_HOST_READ/

A pod with a hostPath volume pointing to a sensitive directory can read host filesystem contents. Checked paths: `/etc`, `/root`, `/home`, `/proc`, `/var/lib/kubelet/pods`.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `hostMountSensitive` | Pod → Node | Pod has a hostPath volume at a sensitive path with a container volume mount |

---

### `lateral_movement_host_mount_kubelet` — Kubelet Config/Credential Access
**File:** `host_mount_kublet.go`

Mounting kubelet directories gives access to node credentials, bootstrap tokens, pod specs, and potentially TLS certificates used for cluster communication. Checked paths: `/var/lib/kubelet`, `/etc/kubernetes`.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `mountedKubelet` | Pod → Node | Pod has a hostPath volume at `/var/lib/kubelet` or `/etc/kubernetes` |

---

### `lateral_movement_pod_mount_service_account` — Auto-Mounted SA Token
**File:** `mount_sa.go`  
**Reference:** https://kubehound.io/reference/attacks/TOKEN_STEAL/

By default (or when `automountServiceAccountToken: true` is explicit for the `default` SA), a pod has a ServiceAccount token mounted at `/var/run/secrets/kubernetes.io/serviceaccount/token`. If an attacker gains code execution in the container, they can use this token to authenticate to the API server with the pod's ServiceAccount identity.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `mountSA` | Pod → ServiceAccount | Pod automounts the SA token: non-default SA unless explicitly disabled, or default SA with explicit `true` |

---

### `cluster` — Persistent Volume Relationships
**File:** `volumes.go`

Contextual edges for storage. Tracks which PVCs are mounted by which pods, and which PVs are bound to which PVCs.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `MountedBy` | PersistentVolumeClaim → Pod | Pod has a volume referencing the PVC name |
| `BoundTo` | PersistentVolume → PersistentVolumeClaim | PV's `claimRef` matches the PVC name and namespace |

---

## Networking (`internal/edges/networking/`)

### `ingress` — Ingress Routing
**File:** `ingress.go`

Maps traffic paths through Ingress objects. `ExternalRoutesTo` edges indicate that an Ingress is reachable from outside the cluster.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `RoutesTo` | Ingress → Service | Ingress backend ref names the service |
| `ExternalRoutesTo` | External → Ingress | Any Ingress exists and an External node is present |

---

### `gateway` — Gateway API Routing
**File:** `gateway.go`

Maps Gateway API traffic paths. Gateways route to route objects; the `ExternalRoutesTo` edge into a Gateway indicates it is internet-facing.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `RoutesTo` | Gateway → HTTPRoute/GRPCRoute/TCPRoute/TLSRoute | Route's `parentRef` names the gateway (matched by name and namespace) |
| `ExternalRoutesTo` | External → Gateway | Any Gateway exists and an External node is present |

---

### `networkpolicy` — NetworkPolicy Application
**File:** `network_policy.go`

Links a NetworkPolicy to the pods it governs via label selector. Used for contextual awareness of what traffic restrictions are in place for a given pod.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `AppliesTo` | NetworkPolicy → Pod | Pod labels satisfy the NetworkPolicy's `podSelector` |

---

### `httproutes`, `grpcroutes`, `tcproutes`, `tlsroutes` — Gateway API Route Backends
**File:** `routes.go`

Maps route objects to their backend Services.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `RoutesTo` | HTTPRoute → Service | Route backendRef names the service |
| `RoutesTo` | GRPCRoute → Service | Same |
| `RoutesTo` | TCPRoute → Service | Same |
| `RoutesTo` | TLSRoute → Service | Same |

---

### `services` — Externally Exposed Services and Service-to-Pod Routing
**File:** `service.go`

Services of type `NodePort` or `LoadBalancer` are accessible from outside the cluster. `LoadBalancer` services include their external IPs in the edge properties. All Services with a non-empty label selector emit `RoutesTo` edges to the pods they select, completing the external path chain: `External → Service → Pod`.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `ExternalRoutesTo` | External → Service | Service type is `NodePort` or `LoadBalancer` |
| `RoutesTo` | Service → Pod | Service has a non-empty `spec.selector` and a pod's labels satisfy it |

---

## Workload (`internal/edges/workload/`)

### `deployment`, `daemonset`, `statefulset`, `job`, `cronjob` — Controller Ownership
**File:** `controllers.go`

Contextual edges that link workload controllers to the pods they manage, resolved by label selector matching.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `ManagedBy` | Deployment → Pod | Pod labels match the Deployment's selector |
| `ManagedBy` | DaemonSet → Pod | Pod labels match the DaemonSet's selector |
| `ManagedBy` | StatefulSet → Pod | Pod labels match the StatefulSet's selector |
| `ManagedBy` | Job → Pod | Pod labels match the Job's selector |
| `ManagedBy` | CronJob → Pod | Pod labels match the CronJob's selector |

---

### `pods` — Pod Scheduling and Secret/ConfigMap Injection
**File:** `pod_config_secret.go`

Covers three relationship types for pods:

**Scheduling:**

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `ScheduledOn` | Pod → Node | Pod has a `nodeName` set |

**Secret injection** (both volume-mount and environment-variable paths):

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `MountedBy` | Secret → Pod | Pod has a `secret` volume referencing the secret name |
| `EnvVars` | Secret → Pod | Pod container has `envFrom.secretRef` referencing the secret name |

---

### `configmap` — ConfigMap Injection
**File:** `pod_config_secret.go`

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `MountedBy` | ConfigMap → Pod | Pod has a `configmap` volume referencing the ConfigMap name |
| `EnvVars` | ConfigMap → Pod | Pod container has `envFrom.configMapRef` referencing the ConfigMap name |

---

### `secret` — Service Account Token Secrets
**File:** `pod_config_secret.go`

Older-style SA token secrets (type `kubernetes.io/service-account-token`) are explicitly linked to the ServiceAccount they belong to.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `SAToken` | Secret → ServiceAccount | Secret type is `kubernetes.io/service-account-token` and the secret name appears in the SA's `.secrets[]` list |

---

## Addons (`internal/edges/addons/`)

### `external_secrets` — External Secrets Operator
**File:** `external_secrets.go`

Maps the ExternalSecrets Operator graph: ExternalSecret objects pull secrets from a backing store (SecretStore or ClusterSecretStore) and sync them into Kubernetes Secrets.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `ManagedBy` | ExternalSecret → SecretStore | ExternalSecret's `storeRef.name` matches a SecretStore in the same namespace |
| `ManagedBy` | ExternalSecret → ClusterSecretStore | ExternalSecret's `storeRef.kind` is `ClusterSecretStore` and the name matches |
| `ManagedBy` | Secret → ExternalSecret | Secret name matches ExternalSecret's `target.name` (or the ExternalSecret's own name as fallback) |

> **Note:** `cert_manager.go` and `istio.go` are present but currently empty stubs.

---

## Aggregates (`internal/edges/aggregates/`)

### `aggregate_contains` — Aggregate Containment
**File:** `contains.go`

Builds the two-level containment hierarchy that makes aggregate-targeted RBAC edges traversable in BloodHound. Without these edges a path like `SA -[SAReadSecret]-> AllSecrets` would be a dead end; `Contains` edges let a query continue from `AllSecrets` down to individual `Secret` nodes.

Two pass types run per execution:

- **Cluster → namespace aggregates**: each cluster-wide aggregate (e.g. `AllSecrets[cluster]`) emits a `Contains` edge to each per-namespace aggregate of the same kind (`AllSecrets[default]`, `AllSecrets[kube-system]`, …).
- **Namespace aggregate → individual resources**: each per-namespace aggregate emits a `Contains` edge to every resource of that kind in the same namespace.

Covers 11 aggregate kinds: `AllPods`, `AllSecrets`, `AllConfigMaps`, `AllServiceAccounts`, `AllDeployments`, `AllDaemonSets`, `AllStatefulSets`, `AllJobs`, `AllCronJobs`, `AllRoles`, `AllClusterRoles`.

`AllClusterRoles` and `AllNodes` are cluster-scoped only (no per-namespace variant). All others have both a cluster-level and a per-namespace node.

| Edge | Source → Target | Trigger |
|------|----------------|---------|
| `Contains` | Cluster aggregate → Namespace aggregate | Both aggregate nodes exist for the same resource kind (applies to all namespace-scoped kinds) |
| `Contains` | Namespace aggregate → Individual resource | A resource of the matching kind exists in that namespace |
| `Contains` | AllClusterRoles → ClusterRole | ClusterRole exists in the cluster |

---

## Adding New Rules

Add rules under a domain package in `internal/edges/<domain>/` and register them explicitly from that package's `Register` function. The top-level `BuildEdges` wiring in `internal/edges/edge_registry.go` calls each domain register function.

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