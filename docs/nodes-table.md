# Node Types — Consolidated Table

| Domain | Kind (BloodHound) | Go File | Scope | Notes |
|--------|-------------------|---------|-------|-------|
| platform | `Namespace` | `platform/namespace.go` | cluster | Metadata-only fetch |
| platform | `Node` | `platform/node.go` | cluster | Kubernetes worker/control-plane node; metadata-only fetch |
| platform | `External` | `platform/external.go` | cluster | Synthetic node for external traffic; always created |
| platform | `AllPods` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `AllSecrets` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `AllConfigMaps` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `AllServiceAccounts` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `AllNodes` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `AllDeployments` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `AllDaemonSets` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `AllStatefulSets` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `AllJobs` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `AllCronJobs` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| rbac | `ServiceAccount` | `rbac/serviceaccount.go` | namespace | |
| rbac | `Role` | `rbac/role.go` | namespace | Namespaced RBAC role |
| rbac | `ClusterRole` | `rbac/clusterrole.go` | cluster | |
| rbac | `RoleBinding` | `rbac/rolebinding.go` | namespace | |
| rbac | `ClusterRoleBinding` | `rbac/clusterrolebinding.go` | cluster | |
| workload | `Pod` | `workload/pod.go` | namespace | Full spec fetch |
| workload | `Deployment` | `workload/deployment.go` | namespace | |
| workload | `DaemonSet` | `workload/daemonset.go` | namespace | |
| workload | `StatefulSet` | `workload/statefulset.go` | namespace | |
| workload | `Job` | `workload/job.go` | namespace | |
| workload | `CronJob` | `workload/cronjob.go` | namespace | |
| workload | `Secret` | `workload/secret.go` | namespace | `data` omitted when `--redacted` |
| workload | `ConfigMap` | `workload/configmap.go` | namespace | |
| networking | `Service` | `networking/service.go` | namespace | |
| networking | `Ingress` | `networking/ingress.go` | namespace | |
| networking | `NetworkPolicy` | `networking/networkpolicy.go` | namespace | |
| networking | `Gateway` | `networking/gateway.go` | namespace | `gateway.networking.k8s.io/v1` and `v1beta1` |
| networking | `HTTPRoute` | `networking/httproute.go` | namespace | `gateway.networking.k8s.io/v1` and `v1beta1` |
| networking | `GRPCRoute` | `networking/grpcroute.go` | namespace | `gateway.networking.k8s.io/v1` and `v1alpha2` |
| networking | `TCPRoute` | `networking/tcproute.go` | namespace | `gateway.networking.k8s.io/v1alpha2` |
| networking | `TLSRoute` | `networking/tlsroute.go` | namespace | `gateway.networking.k8s.io/v1` and `v1alpha2` |
| mounts | `PersistentVolume` | `mounts/pv.go` | cluster | Full spec fetch |
| mounts | `PersistentVolumeClaim` | `mounts/pvc.go` | namespace | Metadata-only fetch |
| addons | `SecretStore` | `addons/external_secrets.go` | namespace | external-secrets operator |
| addons | `ClusterSecretStore` | `addons/external_secrets.go` | cluster | external-secrets operator |
| addons | `ExternalSecret` | `addons/external_secrets.go` | namespace | external-secrets operator |
| addons | `SecurityContextConstraints` | `addons/security_context_constraints.go` | cluster | OpenShift only; builder exists but not yet registered |
