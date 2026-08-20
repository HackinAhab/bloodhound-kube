# Node Types — Consolidated Table

| Domain | Kind (BloodHound) | Go File | Scope | Notes |
|--------|-------------------|---------|-------|-------|
| platform | `BHK_Namespace` | `platform/namespace.go` | cluster | Metadata-only fetch |
| platform | `BHK_Node` | `platform/node.go` | cluster | Kubernetes worker/control-plane node; metadata-only fetch |
| platform | `BHK_External` | `platform/external.go` | cluster | Synthetic node for external traffic; always created |
| platform | `BHK_AllPods` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllSecrets` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllConfigMaps` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllServiceAccounts` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllNodes` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllDeployments` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllDaemonSets` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllStatefulSets` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllJobs` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllCronJobs` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllClusterRoles` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllUsers` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllGroups` | `platform/aggregates.go` | cluster | Aggregate; always created by parser |
| platform | `BHK_AllRoles` | `platform/aggregates.go` | namespace | Aggregate; created per-namespace by parser |
| rbac | `BHK_ServiceAccount` | `rbac/serviceaccount.go` | namespace | |
| rbac | `BHK_User` | `rbac/user.go` | cluster | Created from binding subjects; also carries `BHK_Identity` |
| rbac | `BHK_Group` | `rbac/group.go` | cluster | Created from binding subjects; also carries `BHK_Identity` |
| rbac | `BHK_Role` | `rbac/role.go` | namespace | Namespaced RBAC role |
| rbac | `BHK_ClusterRole` | `rbac/clusterrole.go` | cluster | |
| rbac | `BHK_RoleBinding` | `rbac/rolebinding.go` | namespace | |
| rbac | `BHK_ClusterRoleBinding` | `rbac/clusterrolebinding.go` | cluster | |
| workload | `BHK_Pod` | `workload/pod.go` | namespace | Full spec fetch |
| workload | `BHK_Deployment` | `workload/deployment.go` | namespace | |
| workload | `BHK_DaemonSet` | `workload/daemonset.go` | namespace | |
| workload | `BHK_StatefulSet` | `workload/statefulset.go` | namespace | |
| workload | `BHK_Job` | `workload/job.go` | namespace | |
| workload | `BHK_CronJob` | `workload/cronjob.go` | namespace | |
| workload | `BHK_Secret` | `workload/secret.go` | namespace | `data` omitted when `--redacted` |
| workload | `BHK_ConfigMap` | `workload/configmap.go` | namespace | |
| networking | `BHK_Service` | `networking/service.go` | namespace | |
| networking | `BHK_Ingress` | `networking/ingress.go` | namespace | |
| networking | `BHK_NetworkPolicy` | `networking/networkpolicy.go` | namespace | |
| networking | `BHK_Gateway` | `networking/gateway.go` | namespace | `gateway.networking.k8s.io/v1` and `v1beta1` |
| networking | `BHK_HTTPRoute` | `networking/httproute.go` | namespace | `gateway.networking.k8s.io/v1` and `v1beta1` |
| networking | `BHK_GRPCRoute` | `networking/grpcroute.go` | namespace | `gateway.networking.k8s.io/v1` and `v1alpha2` |
| networking | `BHK_TCPRoute` | `networking/tcproute.go` | namespace | `gateway.networking.k8s.io/v1alpha2` |
| networking | `BHK_TLSRoute` | `networking/tlsroute.go` | namespace | `gateway.networking.k8s.io/v1` and `v1alpha2` |
| mounts | `BHK_PersistentVolume` | `mounts/pv.go` | cluster | Full spec fetch |
| mounts | `BHK_PersistentVolumeClaim` | `mounts/pvc.go` | namespace | Metadata-only fetch |
| addons | `BHK_SecretStore` | `addons/external_secrets.go` | namespace | external-secrets operator |
| addons | `BHK_ClusterSecretStore` | `addons/external_secrets.go` | cluster | external-secrets operator |
| addons | `BHK_ExternalSecret` | `addons/external_secrets.go` | namespace | external-secrets operator |
| addons | `BHK_SecurityContextConstraint` | `addons/security_context_constraints.go` | cluster | OpenShift only; builder exists but not yet registered |
| addons | `BHK_Route` | `addons/route/route.go` | namespace | OpenShift only (`route.openshift.io/v1`); backend refs from `spec.to`/`spec.alternateBackends` |
| addons | `BHK_CiliumNetworkPolicy` | `addons/cilium/cilium_network_policy.go` | namespace | `cilium.io/v2`; unstructured, full spec fetch |
| addons | `BHK_GlobalNetworkPolicy` | `addons/calico/calico_global_network_policy.go` | cluster | `projectcalico.org/v3` and `crd.projectcalico.org/v1`; both groups map to the same builder, deduped at parse time |
| addons | `BHK_Certificate` | `addons/certmanager/cert_manager.go` | namespace | `cert-manager.io/v1` |
| addons | `BHK_Issuer` | `addons/certmanager/cert_manager.go` | namespace | `cert-manager.io/v1`; CA/Vault secret refs only |
| addons | `BHK_ClusterIssuer` | `addons/certmanager/cert_manager.go` | cluster | `cert-manager.io/v1`; CA/Vault secret refs only |
| addons | `BHK_IstioGateway` | `addons/istio/istio.go` | namespace | `networking.istio.io/v1`; named to avoid collision with Gateway API's `BHK_Gateway` |
| addons | `BHK_VirtualService` | `addons/istio/istio.go` | namespace | `networking.istio.io/v1` |
| addons | `BHK_PeerAuthentication` | `addons/istio/istio.go` | namespace | `security.istio.io/v1`; `PERMISSIVE` mode flagged in properties |
| addons | `BHK_AuthorizationPolicy` | `addons/istio/istio.go` | namespace | `security.istio.io/v1`; allow-all (`ALLOW` with no rules) flagged in properties |

## Secondary Types (Cypher Query Helpers)

These are not standalone node kinds — they are additional labels carried by existing nodes to enable cross-type Cypher queries.

| Kind | Applied To | Purpose |
|------|-----------|---------|
| `BHK_Identity` | `BHK_ServiceAccount`, `BHK_User`, `BHK_Group` | `MATCH (n:BHK_Identity)` queries all principal types |
| `BHK_Aggregate` | All `BHK_All*` nodes | `MATCH (n:BHK_Aggregate)` queries all aggregate targets |
