# Edge Rules — Consolidated Table

| Domain | Rule | Edge | Source → Target | Trigger |
|--------|------|------|----------------|---------|
| rbac | `rbac_base` | `RoleBound` | Role/ClusterRole → ServiceAccount | A RoleBinding or ClusterRoleBinding links the role to a ServiceAccount subject |
| rbac | `rbac_impersonate` | `SAImpersonate` | ServiceAccount → ServiceAccount | SA has `impersonate` verb on `serviceaccounts` via a RoleBinding or ClusterRoleBinding |
| rbac | `rbac_impersonate` | `SAImpersonate` | ServiceAccount → AllServiceAccounts | Cluster-scoped binding with wildcard resource access |
| rbac | `rbac_pod_exec` | `PodExec` | ServiceAccount → Pod | SA has `create` on `pods/exec` |
| rbac | `rbac_pod_exec` | `PodExec` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod access |
| rbac | `rbac_pod_debug` | `PodDebug` | ServiceAccount → Pod | SA has `update` on `pods/ephemeralcontainers` |
| rbac | `rbac_read_logs` | `ReadLogs` | ServiceAccount → Pod | SA has `get`/`list`/`watch` on `pods/log` |
| rbac | `rbac_read_logs` | `ReadLogs` | ServiceAccount → AllPods | Cluster-scoped, wildcard pod log access |
| rbac | `rbac_read_secrets` | `SAReadSecret` | ServiceAccount → Secret | SA has `get`/`list`/`watch` on `secrets` |
| rbac | `rbac_read_secrets` | `SAReadSecret` | ServiceAccount → AllSecrets | Cluster-scoped, wildcard secret access |
| rbac | `rbac_read_configmaps` | `ReadConfigMap` | ServiceAccount → ConfigMap | SA has `get`/`list`/`watch` on `configmaps` |
| rbac | `rbac_read_configmaps` | `ReadConfigMap` | ServiceAccount → AllConfigMaps | Cluster-scoped, wildcard configmap access |
| rbac | `rbac_create` | `RBACCreate` | ServiceAccount → Role/ClusterRole | SA has `create` on `rolebindings` (namespaced) or `clusterrolebindings` (cluster-scoped) |
| rbac | `rbac_create_workload` | `WorkloadCreate` | ServiceAccount → Node | SA has `create` on pods, deployments, daemonsets, statefulsets, jobs, or cronjobs |
| rbac | `rbac_patch_workload` | `WorkloadPatch` | ServiceAccount → Pod | SA has `patch`/`update` on `pods` |
| rbac | `rbac_patch_workload` | `WorkloadPatch` | ServiceAccount → Deployment | SA has `patch`/`update` on `deployments` |
| rbac | `rbac_patch_workload` | `WorkloadPatch` | ServiceAccount → DaemonSet | SA has `patch`/`update` on `daemonsets` |
| rbac | `rbac_patch_workload` | `WorkloadPatch` | ServiceAccount → StatefulSet | SA has `patch`/`update` on `statefulsets` |
| rbac | `rbac_patch_workload` | `WorkloadPatch` | ServiceAccount → Job | SA has `patch`/`update` on `jobs` |
| rbac | `rbac_patch_workload` | `WorkloadPatch` | ServiceAccount → CronJob | SA has `patch`/`update` on `cronjobs` |
| rbac | `rbac_node_proxy` | `NodeProxy` | ServiceAccount → Pod | SA has `get` on `nodes/proxy` via a namespaced binding; pod is on a matching node |
| rbac | `rbac_node_proxy` | `NodeProxyRCE` | ServiceAccount → Pod | SA has `get`/`create`/`proxy` on `nodes/proxy` via a cluster-scoped binding |
| security | `capabilities` | `CAP_SYS_ADMIN` | Pod → Node | Container has `CAP_SYS_ADMIN` in `securityContext.capabilities.add` |
| security | `capabilities` | `CAP_NET_ADMIN` | Pod → Node | Container has `CAP_NET_ADMIN` in `securityContext.capabilities.add` |
| security | `capabilities` | `CAP_SYS_MODULE` | Pod → Node | Container has `CAP_SYS_MODULE` in `securityContext.capabilities.add` |
| security | `capabilities` | `CAP_SYS_PTRACE` | Pod → Node | Container has `CAP_SYS_PTRACE` in `securityContext.capabilities.add` |
| security | `capabilities` | `CAP_SYS_RAWIO` | Pod → Node | Container has `CAP_SYS_RAWIO` in `securityContext.capabilities.add` |
| security | `container_escapes` | `CE_PRIV_MOUNT` | Pod → Node | Any container has `privileged: true` |
| security | `container_escapes` | `CE_NSENTER` | Pod → Node | Pod has `hostPID: true` AND a privileged container |
| security | `container_escapes` | `CE_SYS_PTRACE` | Pod → Node | Privileged container, OR (hostPID + CAP_SYS_PTRACE + CAP_SYS_ADMIN) |
| security | `container_escapes` | `CE_UMH_CORE_PATTERN` | Pod → Node | HostPath volume at `/proc`, `/proc/sys`, or `/proc/sys/kernel` with a writable container mount |
| security | `container_escapes` | `MOUNT_CONTAINER_SOCKET` | Pod → Node | HostPath volume path ends in `.sock` |
| security | `container_escapes` | `CE_VAR_LOG_SYMLINK` | Pod → Node | HostPath at `/var/log` or `/var`, container is privileged and not `runAsNonRoot` |
| security | `security_context_constraints` | `EnforcedSCC` | SecurityContextConstraints → Pod | Pod has `openshift.io/scc` annotation matching an SCC name |
| security | `host_ports` | `HostPort` | Node → Pod | Any container in the pod has `hostPort > 0` |
| security | `host_ports` | `ExternalHostPort` | External → Node | Any container in the pod has `hostPort > 0` |
| mounts | `lateral_movement_host_mount_read` | `hostMountSensitive` | Pod → Node | Pod has a hostPath volume at `/etc`, `/root`, `/home`, `/proc`, or `/var/lib/kubelet/pods` with a container mount |
| mounts | `lateral_movement_host_mount_kubelet` | `mountedKubelet` | Pod → Node | Pod has a hostPath volume at `/var/lib/kubelet` or `/etc/kubernetes` |
| mounts | `lateral_movement_pod_mount_service_account` | `mountedSA` | Pod → ServiceAccount | Pod automounts SA token: non-default SA unless explicitly disabled, or default SA with explicit `true` |
| mounts | `cluster` | `MountedBy` | PersistentVolumeClaim → Pod | Pod has a volume referencing the PVC name |
| mounts | `cluster` | `BoundTo` | PersistentVolume → PersistentVolumeClaim | PV's `claimRef` matches the PVC name and namespace |
| networking | `ingress` | `RoutesTo` | Ingress → Service | Ingress backend ref names the service |
| networking | `ingress` | `ExternalRoutesTo` | External → Ingress | Any Ingress exists and an External node is present |
| networking | `gateway` | `RoutesTo` | Gateway → HTTPRoute/GRPCRoute/TCPRoute/TLSRoute | Route's `parentRef` names the gateway (matched by name and namespace) |
| networking | `gateway` | `ExternalRoutesTo` | Gateway → External | Any Gateway exists and an External node is present |
| networking | `networkpolicy` | `AppliesTo` | NetworkPolicy → Pod | Pod labels satisfy the NetworkPolicy's `podSelector` |
| networking | `httproutes` | `RoutesTo` | HTTPRoute → Service | Route backendRef names the service |
| networking | `grpcroutes` | `RoutesTo` | GRPCRoute → Service | Route backendRef names the service |
| networking | `tcproutes` | `RoutesTo` | TCPRoute → Service | Route backendRef names the service |
| networking | `tlsroutes` | `RoutesTo` | TLSRoute → Service | Route backendRef names the service |
| networking | `services` | `ExternalRoutesTo` | External → Service | Service type is `NodePort` or `LoadBalancer` |
| workload | `deployment` | `ManagedBy` | Deployment → Pod | Pod labels match the Deployment's selector |
| workload | `daemonset` | `ManagedBy` | DaemonSet → Pod | Pod labels match the DaemonSet's selector |
| workload | `statefulset` | `ManagedBy` | StatefulSet → Pod | Pod labels match the StatefulSet's selector |
| workload | `job` | `ManagedBy` | Job → Pod | Pod labels match the Job's selector |
| workload | `cronjob` | `ManagedBy` | CronJob → Pod | Pod labels match the CronJob's selector |
| workload | `pods` | `ScheduledOn` | Pod → Node | Pod has a `nodeName` set |
| workload | `pods` | `MountedBy` | Secret → Pod | Pod has a `secret` volume referencing the secret name |
| workload | `pods` | `EnvVars` | Secret → Pod | Pod container has `envFrom.secretRef` referencing the secret name |
| workload | `configmap` | `MountedBy` | ConfigMap → Pod | Pod has a `configmap` volume referencing the ConfigMap name |
| workload | `configmap` | `EnvVars` | ConfigMap → Pod | Pod container has `envFrom.configMapRef` referencing the ConfigMap name |
| workload | `secret` | `SAToken` | Secret → ServiceAccount | Secret type is `kubernetes.io/service-account-token` and the secret name appears in the SA's `.secrets[]` list |
| addons | `external_secrets` | `ManagedBy` | ExternalSecret → SecretStore | ExternalSecret's `storeRef.name` matches a SecretStore in the same namespace |
| addons | `external_secrets` | `ManagedBy` | ExternalSecret → ClusterSecretStore | ExternalSecret's `storeRef.kind` is `ClusterSecretStore` and the name matches |
| addons | `external_secrets` | `ManagedBy` | Secret → ExternalSecret | Secret name matches ExternalSecret's `target.name` (or the ExternalSecret's own name as fallback) |
