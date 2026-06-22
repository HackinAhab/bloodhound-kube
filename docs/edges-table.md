# Edge Rules — Consolidated Table

| Domain | Rule | Edge | Source → Target | Trigger |
|--------|------|------|----------------|---------|
| rbac | `rbac_base` | `BHK_RoleBound` | ServiceAccount → Role/ClusterRole | A RoleBinding or ClusterRoleBinding links the ServiceAccount subject to the role. Edge properties (`bindingKind`, `bindingName`, `bindingNamespace`, `roleKind`, `roleName`, `bindings[]`, `bindingCount`) identify every contributing binding. |
| rbac | `rbac_impersonate` | `BHK_Impersonate` | Identity → ServiceAccount | Identity has `impersonate` verb on `serviceaccounts` via a RoleBinding or ClusterRoleBinding |
| rbac | `rbac_impersonate` | `BHK_Impersonate` | Identity → AllServiceAccounts | Cluster-scoped binding with wildcard ServiceAccount access |
| rbac | `rbac_impersonate` | `BHK_Impersonate` | Identity → User / AllUsers | Identity has `impersonate` on `users` resource (named or wildcard) |
| rbac | `rbac_impersonate` | `BHK_Impersonate` | Identity → Group / AllGroups | Identity has `impersonate` on `groups` resource (named or wildcard) |
| rbac | `rbac_pod_exec` | `BHK_PodExec` | ServiceAccount → Pod | SA has `create` on `pods/exec` |
| rbac | `rbac_pod_exec` | `BHK_PodExec` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod access |
| rbac | `rbac_pod_debug` | `BHK_PodDebug` | ServiceAccount → Pod | SA has `update` on `pods/ephemeralcontainers` |
| rbac | `rbac_read_logs` | `BHK_ReadLogs` | ServiceAccount → Pod | SA has `get`/`list`/`watch` on `pods/log` |
| rbac | `rbac_read_logs` | `BHK_ReadLogs` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod log access |
| rbac | `rbac_read_secrets` | `BHK_ReadSecret` | ServiceAccount → Secret | SA has `get`/`list`/`watch` on `secrets` |
| rbac | `rbac_read_secrets` | `BHK_ReadSecret` | ServiceAccount → AllSecrets | Cluster-scoped, wildcard secret access |
| rbac | `rbac_read_configmaps` | `BHK_ReadConfigMap` | ServiceAccount → ConfigMap | SA has `get`/`list`/`watch` on `configmaps` |
| rbac | `rbac_read_configmaps` | `BHK_ReadConfigMap` | ServiceAccount → AllConfigMaps | Cluster-scoped, wildcard configmap access |
| rbac | `rbac_create` | `BHK_RBACCreate` | ServiceAccount → Role/ClusterRole | SA has `create` on `rolebindings` (namespaced) or `clusterrolebindings` (cluster-scoped) |
| rbac | `rbac_create_workload` | `BHK_WorkloadCreate` | ServiceAccount → Node | SA has `create` on pods, deployments, daemonsets, statefulsets, jobs, or cronjobs |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → Pod | SA has `patch`/`update` on `pods` |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → Deployment | SA has `patch`/`update` on `deployments` |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → DaemonSet | SA has `patch`/`update` on `daemonsets` |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → StatefulSet | SA has `patch`/`update` on `statefulsets` |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → Job | SA has `patch`/`update` on `jobs` |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → CronJob | SA has `patch`/`update` on `cronjobs` |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → AllPods | Cluster-scoped binding with wildcard pod patch access |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → AllDeployments | Cluster-scoped binding with wildcard deployment patch access |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → AllDaemonSets | Cluster-scoped binding with wildcard daemonset patch access |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → AllStatefulSets | Cluster-scoped binding with wildcard statefulset patch access |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → AllJobs | Cluster-scoped binding with wildcard job patch access |
| rbac | `rbac_patch_workload` | `BHK_WorkloadPatch` | ServiceAccount → AllCronJobs | Cluster-scoped binding with wildcard cronjob patch access |
| rbac | `rbac_node_proxy` | `BHK_NodeProxy` | ServiceAccount → Pod | SA has `get` on `nodes/proxy` via a namespaced binding; pod is on a matching node |
| rbac | `rbac_node_proxy` | `BHK_NodeProxyRCE` | ServiceAccount → Pod | SA has `get`/`create`/`proxy` on `nodes/proxy` via a cluster-scoped binding |
| rbac | `rbac_node_proxy` | `BHK_NodeProxyRCE` | ServiceAccount → AllNodes | Cluster-scoped binding with wildcard node access (`all` flag set) |
| rbac | `rbac_pod_portforward` | `BHK_PodPortForward` | ServiceAccount → Pod | SA has `create` on `pods/portforward` |
| rbac | `rbac_pod_portforward` | `BHK_PodPortForward` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod access |
| rbac | `rbac_pod_attach` | `BHK_PodAttach` | ServiceAccount → Pod | SA has `create` on `pods/attach` |
| rbac | `rbac_pod_attach` | `BHK_PodAttach` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod access |
| rbac | `rbac_sa_token_request` | `BHK_SATokenRequest` | ServiceAccount → ServiceAccount | SA has `create` on `serviceaccounts/token` via a RoleBinding |
| rbac | `rbac_sa_token_request` | `BHK_SATokenRequest` | ServiceAccount → AllServiceAccounts | Cluster-scoped, wildcard serviceaccount access |
| rbac | `rbac_escalate_bind` | `BHK_RBACEscalate` | ServiceAccount → Role/ClusterRole | SA has `escalate` on `roles` or `clusterroles` (namespaced binding, or cluster-scoped with named targets) |
| rbac | `rbac_escalate_bind` | `BHK_RBACBind` | ServiceAccount → Role/ClusterRole | SA has `bind` on `roles` or `clusterroles` without `escalate` (namespaced binding, or cluster-scoped with named targets) |
| rbac | `rbac_escalate_bind` | `BHK_RBACEscalate` | ServiceAccount → AllClusterRoles | Cluster-scoped binding with wildcard ClusterRole access (`escalate`) |
| rbac | `rbac_escalate_bind` | `BHK_RBACBind` | ServiceAccount → AllClusterRoles | Cluster-scoped binding with wildcard ClusterRole access (`bind` only) |
| rbac | `rbac_escalate_bind` | `BHK_RBACEscalate` | ServiceAccount → AllRoles | Cluster-scoped binding with wildcard Role access (`escalate`) |
| rbac | `rbac_escalate_bind` | `BHK_RBACBind` | ServiceAccount → AllRoles | Cluster-scoped binding with wildcard Role access (`bind` only) |
| rbac | `rbac_scc_usage` | `BHK_SCCUse` | ServiceAccount → SecurityContextConstraint | SA has `use` on `securitycontextconstraints` via a ClusterRoleBinding (OpenShift only) |
| security | `capabilities` | `BHK_CAP_SYS_ADMIN` | Pod → Node | Container has `BHK_CAP_SYS_ADMIN` in `securityContext.capabilities.add` |
| security | `capabilities` | `BHK_CAP_NET_ADMIN` | Pod → Node | Container has `BHK_CAP_NET_ADMIN` in `securityContext.capabilities.add` |
| security | `capabilities` | `BHK_CAP_SYS_MODULE` | Pod → Node | Container has `BHK_CAP_SYS_MODULE` in `securityContext.capabilities.add` |
| security | `capabilities` | `BHK_CAP_SYS_PTRACE` | Pod → Node | Container has `CAP_SYS_PTRACE` in `securityContext.capabilities.add` |
| security | `capabilities` | `BHK_CAP_SYS_RAWIO` | Pod → Node | Container has `BHK_CAP_SYS_RAWIO` in `securityContext.capabilities.add` |
| security | `container_escapes` | `BHK_CE_PRIV_MOUNT` | Pod → Node | Any container has `privileged: true` |
| security | `container_escapes` | `BHK_CE_NSENTER` | Pod → Node | Pod has `hostPID: true` AND a privileged container |
| security | `container_escapes` | `BHK_CE_SYS_PTRACE` | Pod → Node | Privileged container, OR (hostPID + CAP_SYS_PTRACE + CAP_SYS_ADMIN) |
| security | `container_escapes` | `BHK_CE_UMH_CORE_PATTERN` | Pod → Node | HostPath volume at `/proc`, `/proc/sys`, or `/proc/sys/kernel` with a writable container mount |
| security | `container_escapes` | `BHK_MOUNT_CONTAINER_SOCKET` | Pod → Node | HostPath volume path ends in `.sock` |
| security | `container_escapes` | `BHK_CE_VAR_LOG_SYMLINK` | Pod → Node | HostPath at `/var/log` or `/var`, container is privileged and not `runAsNonRoot` |
| security | `container_escapes` | `BHK_CE_HOST_IPC` | Pod → Node | Pod has `hostIPC: true` AND (privileged container OR `BHK_CAP_SYS_ADMIN`) |
| security | `container_escapes` | `BHK_CE_HOST_NETWORK` | Pod → Node | Pod has `hostNetwork: true` |
| security | `container_escapes` | `BHK_CE_SHARE_PROC_NS` | Pod → Node | Pod has `shareProcessNamespace: true` AND (privileged container OR `CAP_SYS_PTRACE`) |
| security | `security_context_constraints` | `BHK_EnforcedSCC` | SecurityContextConstraint → Pod | Pod has `openshift.io/scc` annotation matching an SCC name |
| security | `host_ports` | `BHK_HostPort` | Node → Pod | Any container in the pod has `hostPort > 0` |
| security | `host_ports` | `BHK_ExternalHostPort` | External → Node | Any container in the pod has `hostPort > 0` |
| mounts | `lateral_movement_host_mount_read` | `BHK_hostMountSensitive` | Pod → Node | Pod has a hostPath volume at `/etc`, `/root`, `/home`, `/proc`, or `/var/lib/kubelet/pods` with a container mount |
| mounts | `lateral_movement_host_mount_kubelet` | `BHK_mountedKubelet` | Pod → Node | Pod has a hostPath volume at `/var/lib/kubelet` or `/etc/kubernetes` |
| mounts | `lateral_movement_pod_mount_service_account` | `BHK_mountSA` | Pod → ServiceAccount | Pod automounts SA token: non-default SA unless explicitly disabled, or default SA with explicit `true` |
| mounts | `cluster` | `BHK_MountedBy` | PersistentVolumeClaim → Pod | Pod has a volume referencing the PVC name |
| mounts | `cluster` | `BHK_BoundTo` | PersistentVolume → PersistentVolumeClaim | PV's `claimRef` matches the PVC name and namespace |
| networking | `ingress` | `BHK_RoutesTo` | Ingress → Service | Ingress backend ref names the service |
| networking | `ingress` | `BHK_ExternalRoutesTo` | External → Ingress | Any Ingress exists and an External node is present |
| networking | `gateway` | `BHK_RoutesTo` | Gateway → HTTPRoute/GRPCRoute/TCPRoute/TLSRoute | Route's `parentRef` names the gateway (matched by name and namespace) |
| networking | `gateway` | `BHK_ExternalRoutesTo` | External → Gateway | Any Gateway exists and an External node is present |
| networking | `networkpolicy` | `BHK_AppliesTo` | NetworkPolicy → Pod | Pod labels satisfy the NetworkPolicy's `podSelector` |
| networking | `httproutes` | `BHK_RoutesTo` | HTTPRoute → Service | Route backendRef names the service |
| networking | `grpcroutes` | `BHK_RoutesTo` | GRPCRoute → Service | Route backendRef names the service |
| networking | `tcproutes` | `BHK_RoutesTo` | TCPRoute → Service | Route backendRef names the service |
| networking | `tlsroutes` | `BHK_RoutesTo` | TLSRoute → Service | Route backendRef names the service |
| networking | `services` | `BHK_ExternalRoutesTo` | External → Service | Service type is `NodePort` or `LoadBalancer` |
| networking | `services` | `BHK_RoutesTo` | Service → Pod | Service has a non-empty `spec.selector` matching pod labels |
| workload | `deployment` | `BHK_ManagedBy` | Deployment → Pod | Pod labels match the Deployment's selector |
| workload | `daemonset` | `BHK_ManagedBy` | DaemonSet → Pod | Pod labels match the DaemonSet's selector |
| workload | `statefulset` | `BHK_ManagedBy` | StatefulSet → Pod | Pod labels match the StatefulSet's selector |
| workload | `job` | `BHK_ManagedBy` | Job → Pod | Pod labels match the Job's selector |
| workload | `cronjob` | `BHK_ManagedBy` | CronJob → Pod | Pod labels match the CronJob's selector |
| workload | `pods` | `BHK_ScheduledOn` | Pod → Node | Pod has a `nodeName` set |
| workload | `pods` | `BHK_MountedBy` | Secret → Pod | Pod has a `secret` volume referencing the secret name |
| workload | `pods` | `BHK_EnvVars` | Secret → Pod | Pod container has `envFrom.secretRef` referencing the secret name |
| workload | `configmap` | `BHK_MountedBy` | ConfigMap → Pod | Pod has a `configmap` volume referencing the ConfigMap name |
| workload | `configmap` | `BHK_EnvVars` | ConfigMap → Pod | Pod container has `envFrom.configMapRef` referencing the ConfigMap name |
| workload | `secret` | `BHK_SAToken` | Secret → ServiceAccount | Secret type is `kubernetes.io/service-account-token` and the secret name appears in the SA's `.secrets[]` list |
| addons | `external_secrets` | `BHK_ManagedBy` | ExternalSecret → SecretStore | ExternalSecret's `storeRef.name` matches a SecretStore in the same namespace |
| addons | `external_secrets` | `BHK_ManagedBy` | ExternalSecret → ClusterSecretStore | ExternalSecret's `storeRef.kind` is `BHK_ClusterSecretStore` and the name matches |
| addons | `external_secrets` | `BHK_ManagedBy` | Secret → ExternalSecret | Secret name matches ExternalSecret's `target.name` (or the ExternalSecret's own name as fallback) |
| aggregates | `aggregate_contains` | `BHK_Contains` | Cluster aggregate → Namespace aggregate | Both aggregate nodes exist for the same resource kind (namespace-scoped kinds only) |
| aggregates | `aggregate_contains` | `BHK_Contains` | Namespace aggregate → Individual resource | A resource of the matching kind exists in that namespace |
| aggregates | `aggregate_contains` | `BHK_Contains` | AllClusterRoles → ClusterRole | ClusterRole exists in the cluster |
