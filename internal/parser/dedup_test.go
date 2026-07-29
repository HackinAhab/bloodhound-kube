//go:build !no_addons && !no_calico

package parser

import (
	"strings"
	"testing"

	"bloodhound-kube/internal/model"
)

// A GlobalNetworkPolicy served under both Calico API groups
// (crd.projectcalico.org/v1 storage CRD + projectcalico.org/v3 canonical
// apiserver) must collapse to a single typed BHK_GlobalNetworkPolicy node —
// never a duplicate generic BHK_GlobalNetworkPolicy:projectcalico.org node —
// regardless of which representation the collector wrote first.

const gnpV1JSONL = `{"apiVersion":"crd.projectcalico.org/v1","kind":"GlobalNetworkPolicy","metadata":{"name":"allow-lan"},"spec":{"selector":"role == 'worker'","types":["Ingress"]}}`

const gnpV3JSONL = `{"apiVersion":"projectcalico.org/v3","kind":"GlobalNetworkPolicy","metadata":{"name":"allow-lan"},"spec":{"selector":"role == 'worker'","types":["Ingress"]}}`

func countKind(nodes []model.BloodHoundNode, kind string) int {
	n := 0
	for _, node := range nodes {
		for _, k := range node.Kinds {
			if k == kind {
				n++
			}
		}
	}
	return n
}

func TestParse_GlobalNetworkPolicyDualGroupDedup(t *testing.T) {
	cases := []struct {
		name  string
		jsonl string
	}{
		{"v1-then-v3", gnpV1JSONL + "\n" + gnpV3JSONL + "\n"},
		{"v3-then-v1", gnpV3JSONL + "\n" + gnpV1JSONL + "\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nodes, core, _, err := createNodesAndCoreFactsFromReader(strings.NewReader(tc.jsonl), true)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			if got := countKind(nodes, "BHK_GlobalNetworkPolicy"); got != 1 {
				t.Fatalf("expected exactly 1 BHK_GlobalNetworkPolicy node, got %d (%v)", got, nodes)
			}
			if got := countKind(nodes, "BHK_GlobalNetworkPolicy:projectcalico.org"); got != 0 {
				t.Fatalf("expected no generic BHK_GlobalNetworkPolicy:projectcalico.org node, got %d", got)
			}
			if got := len(core.Cluster.GlobalNetworkPolicies); got != 1 {
				t.Fatalf("expected exactly 1 GlobalNetworkPolicy CoreFact, got %d", got)
			}
		})
	}
}

// A generic node written before its typed twin must be replaced by the typed
// result (typed-wins), leaving the canonical typed node in place.
func TestParse_TypedWinsOverEarlierGeneric(t *testing.T) {
	// An unregistered kind stays generic; a registered kind (GlobalNetworkPolicy)
	// wins. Feed the generic-producing group first to exercise replacement.
	jsonl := gnpV3JSONL + "\n" + gnpV1JSONL + "\n"
	nodes, _, _, err := createNodesAndCoreFactsFromReader(strings.NewReader(jsonl), true)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	for _, node := range nodes {
		for _, k := range node.Kinds {
			if k == "BHK_GlobalNetworkPolicy" {
				if node.Properties["resource_type"] == "generic" {
					t.Fatalf("expected typed node to win, but node is generic: %v", node.Properties)
				}
			}
		}
	}
}
