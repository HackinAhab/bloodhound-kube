//go:build !no_addons && !no_cert_manager

package certmanager

import (
	"testing"

	"bloodhound-kube/internal/edges/framework"
	"bloodhound-kube/internal/model"
	nodecertmanager "bloodhound-kube/internal/nodes/addons/certmanager"
	nodefw "bloodhound-kube/internal/nodes/framework"
	"bloodhound-kube/internal/nodes/workload"
)

// base/newCore/ensureNamespace/hasEdge are duplicated in each addon edge-rule
// test package (see also edges/addons/cilium, edges/addons/calico) —
// unexported test helpers can't be shared across packages, and a shared
// test-support package would be overkill for four tiny functions.

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

func TestCertManagerEdgesRule_CertificateToSecretAndIssuer(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Certificates = append(ns.Certificates, nodecertmanager.Certificate{
		GraphNodeBase: base("BHK_Certificate", "ns1", "web-cert"),
		SecretName:    "web-tls",
		IssuerRefName: "ca-issuer",
		IssuerRefKind: "Issuer",
	})
	ns.Issuers = append(ns.Issuers, nodecertmanager.Issuer{
		GraphNodeBase: base("BHK_Issuer", "ns1", "ca-issuer"),
		CASecretName:  "ca-key-pair",
	})
	ns.Secrets = append(ns.Secrets,
		workload.Secret{GraphNodeBase: base("BHK_Secret", "ns1", "web-tls")},
		workload.Secret{GraphNodeBase: base("BHK_Secret", "ns1", "ca-key-pair")},
	)

	ctx := framework.NewContext(core)
	edges := certManagerEdgesRule{}.Apply(ctx)

	certID := nodefw.BuildID("BHK_Certificate", "ns1", "web-cert")
	issuerID := nodefw.BuildID("BHK_Issuer", "ns1", "ca-issuer")
	tlsSecretID := nodefw.BuildID("BHK_Secret", "ns1", "web-tls")
	caSecretID := nodefw.BuildID("BHK_Secret", "ns1", "ca-key-pair")

	if !hasEdge(edges, certID, tlsSecretID, "BHK_ManagedBy") {
		t.Fatalf("expected Certificate -> Secret (issued TLS secret) edge, got %v", edges)
	}
	if !hasEdge(edges, certID, issuerID, "BHK_ManagedBy") {
		t.Fatalf("expected Certificate -> Issuer edge, got %v", edges)
	}
	if !hasEdge(edges, issuerID, caSecretID, "BHK_ManagedBy") {
		t.Fatalf("expected Issuer -> Secret (CA signing key) edge — this is the escalation path, got %v", edges)
	}
}

func TestCertManagerEdgesRule_CertificateToClusterIssuer(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Certificates = append(ns.Certificates, nodecertmanager.Certificate{
		GraphNodeBase: base("BHK_Certificate", "ns1", "web-cert"),
		IssuerRefName: "cluster-ca",
		IssuerRefKind: "ClusterIssuer",
	})
	core.Cluster.ClusterIssuers = append(core.Cluster.ClusterIssuers, nodecertmanager.ClusterIssuer{
		GraphNodeBase: base("BHK_ClusterIssuer", "", "cluster-ca"),
	})

	ctx := framework.NewContext(core)
	edges := certManagerEdgesRule{}.Apply(ctx)

	certID := nodefw.BuildID("BHK_Certificate", "ns1", "web-cert")
	clusterIssuerID := nodefw.BuildID("BHK_ClusterIssuer", "", "cluster-ca")

	if !hasEdge(edges, certID, clusterIssuerID, "BHK_ManagedBy") {
		t.Fatalf("expected Certificate -> ClusterIssuer edge, got %v", edges)
	}
}

func TestCertManagerEdgesRule_NoRefsProducesNoEdges(t *testing.T) {
	core := newCore()
	ns := ensureNamespace(core, "ns1")
	ns.Certificates = append(ns.Certificates, nodecertmanager.Certificate{
		GraphNodeBase: base("BHK_Certificate", "ns1", "orphan-cert"),
	})

	ctx := framework.NewContext(core)
	edges := certManagerEdgesRule{}.Apply(ctx)

	if len(edges) != 0 {
		t.Fatalf("expected no edges for a Certificate with no secretName/issuerRef, got %v", edges)
	}
}
