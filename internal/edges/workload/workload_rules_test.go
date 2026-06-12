package workload

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/platform"
	rbacnodes "bloodhound-kube/internal/nodes/rbac"
	"bloodhound-kube/internal/nodes/workload"
)

func base(kind, namespace, name string) nodefw.GraphNodeBase {
	return nodefw.NewGraphNodeBase(kind, namespace, name, nil, nil)
}

func newCore() *model.CoreFacts {
	return &model.CoreFacts{
		Namespaces: map[string]*model.Namespace{},
		Cluster:    &model.Cluster{},
	}
}

func ensureNamespace(core *model.CoreFacts, namespace string) *model.Namespace {
	if core.Namespaces[namespace] == nil {
		core.Namespaces[namespace] = &model.Namespace{}
	}
	return core.Namespaces[namespace]
}

func hasEdge(edges []model.BloodHoundEdge, startID, endID, kind string) bool {
	for _, edge := range edges {
		if edge.Kind == kind && edge.Start.Value == startID && edge.End.Value == endID {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// deploymentEdgesRule
// ---------------------------------------------------------------------------

func TestDeploymentEdgesRuleManagedBy(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Deployments = append(ns.Deployments, workload.Deployment{
		GraphNodeBase:  base("Deployment", "ns1", "my-deploy"),
		SelectorLabels: map[string]string{"app": "web"},
	})
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-a", map[string]any{"app": "web"}, nil)},
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-b", map[string]any{"app": "other"}, nil)},
	)

	ctx := framework.NewContext(core)
	edges := deploymentEdgesRule{}.Apply(ctx)
	deployID := nodefw.BuildID("Deployment", "ns1", "my-deploy")
	if !hasEdge(edges, deployID, nodefw.BuildID("Pod", "ns1", "pod-a"), "ManagedBy") {
		t.Fatalf("missing ManagedBy edge to pod-a")
	}
	if hasEdge(edges, deployID, nodefw.BuildID("Pod", "ns1", "pod-b"), "ManagedBy") {
		t.Fatalf("unexpected ManagedBy edge to pod-b (label mismatch)")
	}
}

func TestDeploymentEdgesRuleEmptySelectorSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Deployments = append(ns.Deployments, workload.Deployment{
		GraphNodeBase:  base("Deployment", "ns1", "no-selector"),
		SelectorLabels: nil,
	})
	ns.Pods = append(ns.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-a")})

	ctx := framework.NewContext(core)
	edges := deploymentEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for deployment with empty selector, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// daemonSetEdgesRule
// ---------------------------------------------------------------------------

func TestDaemonSetEdgesRuleManagedBy(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.DaemonSets = append(ns.DaemonSets, workload.DaemonSetCore{
		GraphNodeBase:  base("DaemonSet", "ns1", "my-ds"),
		SelectorLabels: map[string]string{"tier": "node"},
	})
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-a", map[string]any{"tier": "node"}, nil)},
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-b", map[string]any{"tier": "app"}, nil)},
	)

	ctx := framework.NewContext(core)
	edges := daemonSetEdgesRule{}.Apply(ctx)
	dsID := nodefw.BuildID("DaemonSet", "ns1", "my-ds")
	if !hasEdge(edges, dsID, nodefw.BuildID("Pod", "ns1", "pod-a"), "ManagedBy") {
		t.Fatalf("missing ManagedBy edge to pod-a")
	}
	if hasEdge(edges, dsID, nodefw.BuildID("Pod", "ns1", "pod-b"), "ManagedBy") {
		t.Fatalf("unexpected ManagedBy edge to pod-b")
	}
}

func TestDaemonSetEdgesRuleEmptySelectorSkipped(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.DaemonSets = append(ns.DaemonSets, workload.DaemonSetCore{
		GraphNodeBase: base("DaemonSet", "ns1", "no-selector"),
	})
	ns.Pods = append(ns.Pods, workload.Pod{GraphNodeBase: base("Pod", "ns1", "pod-a")})

	ctx := framework.NewContext(core)
	edges := daemonSetEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no edges for daemonset with empty selector, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// statefulSetEdgesRule
// ---------------------------------------------------------------------------

func TestStatefulSetEdgesRuleManagedBy(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.StatefulSets = append(ns.StatefulSets, workload.StatefulSetCore{
		GraphNodeBase:  base("StatefulSet", "ns1", "my-ss"),
		SelectorLabels: map[string]string{"db": "pg"},
	})
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-a", map[string]any{"db": "pg"}, nil)},
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-b", map[string]any{"db": "mysql"}, nil)},
	)

	ctx := framework.NewContext(core)
	edges := statefulSetEdgesRule{}.Apply(ctx)
	ssID := nodefw.BuildID("StatefulSet", "ns1", "my-ss")
	if !hasEdge(edges, ssID, nodefw.BuildID("Pod", "ns1", "pod-a"), "ManagedBy") {
		t.Fatalf("missing ManagedBy edge to pod-a")
	}
	if hasEdge(edges, ssID, nodefw.BuildID("Pod", "ns1", "pod-b"), "ManagedBy") {
		t.Fatalf("unexpected ManagedBy edge to pod-b")
	}
}

// ---------------------------------------------------------------------------
// jobEdgesRule
// ---------------------------------------------------------------------------

func TestJobEdgesRuleManagedBy(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Jobs = append(ns.Jobs, workload.Job{
		GraphNodeBase:  base("Job", "ns1", "my-job"),
		SelectorLabels: map[string]string{"job-name": "my-job"},
	})
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-a", map[string]any{"job-name": "my-job"}, nil)},
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-b", map[string]any{"job-name": "other"}, nil)},
	)

	ctx := framework.NewContext(core)
	edges := jobEdgesRule{}.Apply(ctx)
	jobID := nodefw.BuildID("Job", "ns1", "my-job")
	if !hasEdge(edges, jobID, nodefw.BuildID("Pod", "ns1", "pod-a"), "ManagedBy") {
		t.Fatalf("missing ManagedBy edge to pod-a")
	}
	if hasEdge(edges, jobID, nodefw.BuildID("Pod", "ns1", "pod-b"), "ManagedBy") {
		t.Fatalf("unexpected ManagedBy edge to pod-b")
	}
}

// ---------------------------------------------------------------------------
// cronJobEdgesRule
// ---------------------------------------------------------------------------

func TestCronJobEdgesRuleManagedBy(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.CronJobs = append(ns.CronJobs, workload.CronJob{
		GraphNodeBase:  base("CronJob", "ns1", "my-cron"),
		SelectorLabels: map[string]string{"cron": "batch"},
	})
	ns.Pods = append(ns.Pods,
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-a", map[string]any{"cron": "batch"}, nil)},
		workload.Pod{GraphNodeBase: nodefw.NewGraphNodeBase("Pod", "ns1", "pod-b", map[string]any{"cron": "other"}, nil)},
	)

	ctx := framework.NewContext(core)
	edges := cronJobEdgesRule{}.Apply(ctx)
	cronID := nodefw.BuildID("CronJob", "ns1", "my-cron")
	if !hasEdge(edges, cronID, nodefw.BuildID("Pod", "ns1", "pod-a"), "ManagedBy") {
		t.Fatalf("missing ManagedBy edge to pod-a")
	}
	if hasEdge(edges, cronID, nodefw.BuildID("Pod", "ns1", "pod-b"), "ManagedBy") {
		t.Fatalf("unexpected ManagedBy edge to pod-b")
	}
}

// ---------------------------------------------------------------------------
// podEdgesRule (ScheduledOn + secret volume MountedBy + EnvVars)
// ---------------------------------------------------------------------------

func TestPodEdgesRuleScheduledOn(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("Pod", "ns1", "my-pod"),
		NodeName:      "node-1",
	})
	core.Cluster.Nodes = append(core.Cluster.Nodes,
		platform.Node{GraphNodeBase: base("Node", "", "node-1")},
	)

	ctx := framework.NewContext(core)
	edges := podEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("Pod", "ns1", "my-pod"), nodefw.BuildID("Node", "", "node-1"), "ScheduledOn") {
		t.Fatalf("missing ScheduledOn edge to node-1")
	}
}

func TestPodEdgesRuleSecretVolumeMountedBy(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Secrets = append(ns.Secrets,
		workload.Secret{GraphNodeBase: base("Secret", "ns1", "my-secret")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("Pod", "ns1", "my-pod"),
		Volumes: []workload.VolumeDetail{
			{Name: "sec-vol", Type: "secret", SecretName: "my-secret"},
		},
	})

	ctx := framework.NewContext(core)
	edges := podEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("Secret", "ns1", "my-secret"), nodefw.BuildID("Pod", "ns1", "my-pod"), "MountedBy") {
		t.Fatalf("missing MountedBy edge from secret to pod")
	}
}

func TestPodEdgesRuleSecretEnvFrom(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Secrets = append(ns.Secrets,
		workload.Secret{GraphNodeBase: base("Secret", "ns1", "env-secret")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("Pod", "ns1", "my-pod"),
		Containers: []workload.Container{
			{EnvFrom: []workload.EnvFromSource{{SecretRef: &workload.NamedObjectRef{Name: "env-secret"}}}},
		},
	})

	ctx := framework.NewContext(core)
	edges := podEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("Secret", "ns1", "env-secret"), nodefw.BuildID("Pod", "ns1", "my-pod"), "EnvVars") {
		t.Fatalf("missing EnvVars edge from secret via envFrom")
	}
}

func TestPodEdgesRuleSecretEnvValueFrom(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Secrets = append(ns.Secrets,
		workload.Secret{GraphNodeBase: base("Secret", "ns1", "key-secret")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("Pod", "ns1", "my-pod"),
		Containers: []workload.Container{
			{Env: []workload.EnvVar{
				{ValueRef: &workload.EnvVarValueRef{SecretRef: &workload.NamedObjectRef{Name: "key-secret"}}},
			}},
		},
	})

	ctx := framework.NewContext(core)
	edges := podEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("Secret", "ns1", "key-secret"), nodefw.BuildID("Pod", "ns1", "my-pod"), "EnvVars") {
		t.Fatalf("missing EnvVars edge from secret via env valueFrom")
	}
}

// ---------------------------------------------------------------------------
// configMapEdgesRule
// ---------------------------------------------------------------------------

func TestConfigMapEdgesRuleMountedBy(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ConfigMaps = append(ns.ConfigMaps,
		workload.ConfigMap{GraphNodeBase: base("ConfigMap", "ns1", "my-cm")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("Pod", "ns1", "my-pod"),
		Volumes: []workload.VolumeDetail{
			{Name: "cm-vol", Type: "configmap", ConfigMapName: "my-cm"},
		},
	})

	ctx := framework.NewContext(core)
	edges := configMapEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("ConfigMap", "ns1", "my-cm"), nodefw.BuildID("Pod", "ns1", "my-pod"), "MountedBy") {
		t.Fatalf("missing MountedBy edge from configmap to pod")
	}
}

func TestConfigMapEdgesRuleEnvFrom(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ConfigMaps = append(ns.ConfigMaps,
		workload.ConfigMap{GraphNodeBase: base("ConfigMap", "ns1", "env-cm")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("Pod", "ns1", "my-pod"),
		Containers: []workload.Container{
			{EnvFrom: []workload.EnvFromSource{{ConfigMapRef: &workload.NamedObjectRef{Name: "env-cm"}}}},
		},
	})

	ctx := framework.NewContext(core)
	edges := configMapEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("ConfigMap", "ns1", "env-cm"), nodefw.BuildID("Pod", "ns1", "my-pod"), "EnvVars") {
		t.Fatalf("missing EnvVars edge from configmap via envFrom")
	}
}

func TestConfigMapEdgesRuleEnvValueFrom(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.ConfigMaps = append(ns.ConfigMaps,
		workload.ConfigMap{GraphNodeBase: base("ConfigMap", "ns1", "key-cm")},
	)
	ns.Pods = append(ns.Pods, workload.Pod{
		GraphNodeBase: base("Pod", "ns1", "my-pod"),
		Containers: []workload.Container{
			{Env: []workload.EnvVar{
				{ValueRef: &workload.EnvVarValueRef{ConfigMapRef: &workload.NamedObjectRef{Name: "key-cm"}}},
			}},
		},
	})

	ctx := framework.NewContext(core)
	edges := configMapEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("ConfigMap", "ns1", "key-cm"), nodefw.BuildID("Pod", "ns1", "my-pod"), "EnvVars") {
		t.Fatalf("missing EnvVars edge from configmap via env valueFrom")
	}
}

// ---------------------------------------------------------------------------
// secretEdgesRule (SAToken)
// ---------------------------------------------------------------------------

func TestSecretEdgesRuleSAToken(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Secrets = append(ns.Secrets,
		workload.Secret{
			GraphNodeBase: base("Secret", "ns1", "sa-token-secret"),
			SecretType:    "kubernetes.io/service-account-token",
		},
		workload.Secret{
			GraphNodeBase: base("Secret", "ns1", "opaque-secret"),
			SecretType:    "Opaque",
		},
	)
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbacnodes.ServiceAccount{
			GraphNodeBase: base("ServiceAccount", "ns1", "my-sa"),
			Secrets:       []string{"sa-token-secret"},
		},
	)

	ctx := framework.NewContext(core)
	edges := secretEdgesRule{}.Apply(ctx)
	if !hasEdge(edges, nodefw.BuildID("Secret", "ns1", "sa-token-secret"), nodefw.BuildID("ServiceAccount", "ns1", "my-sa"), "SAToken") {
		t.Fatalf("missing SAToken edge from service-account-token secret to SA")
	}
	if hasEdge(edges, nodefw.BuildID("Secret", "ns1", "opaque-secret"), nodefw.BuildID("ServiceAccount", "ns1", "my-sa"), "SAToken") {
		t.Fatalf("unexpected SAToken edge from opaque secret")
	}
}

func TestSecretEdgesRuleSATokenNotLinkedToWrongSA(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Secrets = append(ns.Secrets,
		workload.Secret{
			GraphNodeBase: base("Secret", "ns1", "sa-token-secret"),
			SecretType:    "kubernetes.io/service-account-token",
		},
	)
	ns.ServiceAccounts = append(ns.ServiceAccounts,
		rbacnodes.ServiceAccount{
			GraphNodeBase: base("ServiceAccount", "ns1", "other-sa"),
			Secrets:       []string{"different-secret"},
		},
	)

	ctx := framework.NewContext(core)
	edges := secretEdgesRule{}.Apply(ctx)
	if len(edges) != 0 {
		t.Fatalf("expected no SAToken edges when secret not in SA.Secrets list, got %d", len(edges))
	}
}
