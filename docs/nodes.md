# Nodes
Nodes represent entities in the Kubernetes cluster. For example, a pod, a service account. 

## Aggregate Nodes
Aggregate nodes represent a collection of nodes that would be not useful to display individually. For example, AllPods is an aggregate node that represents all pods in the cluster. This is useful because there can be thousands of pods in a cluster, and displaying edges to all of them would be overwhelming, and likely to cause neo4j to explode. The following aggregate nodes are currently implmemented:
- AllPods
    - This is to support a variety of RBAC edges that allow access to pods, such as exec, logs, debug, and if a service account has any widespread access this explodes the graph very quickly. 
- AllDeployments
- AllStatefulSets
- AllDaemonSets
    - These are to support the WorkloadPatch edge, which is a common edge for builtin service accounts for the object controllers, but is worth noting if other service accounts have this access as well. 
- AllSecrets
    - This is to support the SAReadSecrets edge, which can be common for secrets managed by external secrets operators, and is extremely noisy to display edges to every single secret in the cluster
- AllServiceAccounts
    - This is to support the SAImpersonate edge, because it is common for some service accounts to have the ability to impersonate all service accounts in the cluster, and is not always a vulnerable configuration, but is worth noting when it is present.