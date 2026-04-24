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
- AllConfigMaps
    - This is to support the SAReadConfigMap edge when cluster-wide ConfigMap read access exists, avoiding one edge per ConfigMap.
- AllServiceAccounts
    - This is to support the SAImpersonate edge, because it is common for some service accounts to have the ability to impersonate all service accounts in the cluster, and is not always a vulnerable configuration, but is worth noting when it is present.

## Object types not covered at this time
The following object types are not currently covered by BloodHound-Kube, but may be added in a future release:
- `Subject` nodes for anything besides a Service Account.
- `TCPRoute` and `TLSRoute` nodes for Gateway API.
- `CronJob` nodes for batch/v1.
- `Job` nodes for batch/v1.
- `ListenerSets` and `ListenerSetGroups` for Gateway API.

## Add new nodes
#TODO: update docs for building node parsers. 
To add a new node parser, add a Go file that registers a builder in `internal/nodes` and rebuild. Example:

```go
package nodes

type MyKindNode struct{
	GraphNodeBase
	MyProperty string
}

func init() {
	Register("MyKind", BuildMyKindNode)
}

func BuildMyKindNode(resource map[string]any) (BuildResult, bool) {
	metadata := GetMap(resource, "metadata")
	name := GetString(metadata, "name")
	if name == "" {
		return BuildResult{}, false
	}
	namespace := GetString(metadata, "namespace")
	labelsMap := GetMap(metadata, "labels")
	annotationsMap := GetMap(metadata, "annotations")

	spec := GetMap(resource, "spec")
	specialProperty := MyHelperFunction(spec)

	properties := map[string]any{
		"name":         name,
		"namespace":    namespace,
		"labels":       MapToSortedList(labelsMap),
		"annotations":  MapToSortedList(annotationsMap),
		"specialProperty": specialProperty,
	}

	core := CoreEntry{
		Namespace: namespace,
		Cluster:   false,
		Data: SecretStore{
			GraphNodeBase: GraphNodeBase{
				ID:             BuildID("MyKind", namespace, name),
				Kinds:          []string{"MyKind"},
				Name:           name,
				Namespace:      namespace,
				LabelsMap:      labelsMap,
				AnnotationsMap: annotationsMap,
			},
			SpecialProperty: specialProperty,
		},
	}

	return BuildResult{
		Node: NodeResult{
			ID:         BuildID("MyKind", namespace, name),
			Kinds:      []string{"MyKind"},
			Properties: properties,
		},
		Core: []CoreEntry{core},
	}, true
}

func MyHelperFunction(spec map[string]any) string {
	// Extract special property from spec
	return ""
}
```
