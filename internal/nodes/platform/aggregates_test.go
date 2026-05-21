package platform

import (
	"testing"

	fw "bloodhound-kube/internal/nodes/framework"
)

// TestBuildNamespaceAggregate_IDFormat asserts the canonical ID format for
// every namespaced aggregate flavor: "<Kind>:<namespace>:<Kind>". The
// namespace name lives in the middle segment to mirror the standard
// kind:ns:name pattern used elsewhere.
func TestBuildNamespaceAggregate_IDFormat(t *testing.T) {
	cases := []struct {
		name      string
		kind      string
		namespace string
		build     func(string) fw.BuildResult
	}{
		{"AllPodsNS", "AllPods", "prod", BuildAllPodsNS},
		{"AllSecretsNS", "AllSecrets", "prod", BuildAllSecretsNS},
		{"AllConfigMapsNS", "AllConfigMaps", "prod", BuildAllConfigMapsNS},
		{"AllServiceAccountsNS", "AllServiceAccounts", "prod", BuildAllServiceAccountsNS},
		{"AllDeploymentsNS", "AllDeployments", "prod", BuildAllDeploymentsNS},
		{"AllDaemonSetsNS", "AllDaemonSets", "prod", BuildAllDaemonSetsNS},
		{"AllStatefulSetsNS", "AllStatefulSets", "prod", BuildAllStatefulSetsNS},
		{"AllJobsNS", "AllJobs", "prod", BuildAllJobsNS},
		{"AllCronJobsNS", "AllCronJobs", "prod", BuildAllCronJobsNS},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.build(tc.namespace)

			wantID := tc.kind + ":" + tc.namespace + ":" + tc.kind
			if result.Node.ID != wantID {
				t.Fatalf("ID = %q, want %q", result.Node.ID, wantID)
			}
			if len(result.Node.Kinds) != 1 || result.Node.Kinds[0] != tc.kind {
				t.Fatalf("Kinds = %v, want [%q]", result.Node.Kinds, tc.kind)
			}
			if got, want := result.Node.Properties["name"], tc.kind; got != want {
				t.Fatalf("Properties[name] = %v, want %v", got, want)
			}
			if got, want := result.Node.Properties["namespace"], tc.namespace; got != want {
				t.Fatalf("Properties[namespace] = %v, want %v", got, want)
			}
			if len(result.Core) != 1 {
				t.Fatalf("expected 1 core entry, got %d", len(result.Core))
			}
			entry := result.Core[0]
			if entry.Cluster {
				t.Fatalf("namespaced aggregate must have Cluster=false")
			}
			if entry.Namespace != tc.namespace {
				t.Fatalf("CoreEntry.Namespace = %q, want %q", entry.Namespace, tc.namespace)
			}
			if entry.Data == nil {
				t.Fatalf("CoreEntry.Data must not be nil")
			}
		})
	}
}

// TestBuildClusterAggregate_BehaviorUnchanged locks in the existing
// cluster-aggregate format so that adding the namespaced variant doesn't
// regress queries that already match (n:AllSecrets) cluster-wide.
func TestBuildClusterAggregate_BehaviorUnchanged(t *testing.T) {
	result := BuildAllSecrets()

	if result.Node.ID != "AllSecrets:AllSecrets" {
		t.Fatalf("cluster ID = %q, want %q", result.Node.ID, "AllSecrets:AllSecrets")
	}
	if got, want := result.Node.Properties["namespace"], ""; got != want {
		t.Fatalf("cluster namespace property = %v, want empty string", got)
	}
	if len(result.Core) != 1 || !result.Core[0].Cluster {
		t.Fatalf("cluster aggregate must have Cluster=true")
	}
}

// TestNamespacedAndClusterAggregates_AreDistinctNodes asserts that the
// namespaced and cluster flavors of the same kind produce distinct IDs.
// This is the core invariant enabling per-namespace aggregates to coexist
// with the cluster aggregate without collision.
func TestNamespacedAndClusterAggregates_AreDistinctNodes(t *testing.T) {
	cluster := BuildAllSecrets()
	ns := BuildAllSecretsNS("prod")

	if cluster.Node.ID == ns.Node.ID {
		t.Fatalf("cluster and namespaced aggregates must have distinct IDs (got %q for both)", cluster.Node.ID)
	}
	// Same kinds label so a single Cypher query keeps matching both flavors.
	if len(cluster.Node.Kinds) != len(ns.Node.Kinds) || cluster.Node.Kinds[0] != ns.Node.Kinds[0] {
		t.Fatalf("cluster and namespaced aggregates must share the same Kinds label, got cluster=%v ns=%v",
			cluster.Node.Kinds, ns.Node.Kinds)
	}
}
