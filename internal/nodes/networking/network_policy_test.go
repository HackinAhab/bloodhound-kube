package networking

import (
	"slices"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func policyTypeIngress() networkingv1.PolicyType { return networkingv1.PolicyTypeIngress }
func policyTypeEgress() networkingv1.PolicyType  { return networkingv1.PolicyTypeEgress }

func strContains(slice []string, s string) bool {
	return slices.Contains(slice, s)
}

func TestBuildNetworkPolicyNode_BasicProperties(t *testing.T) {
	proto := corev1.ProtocolTCP
	port80 := intstr.FromInt(80)
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-web", Namespace: "prod"},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &proto, Port: &port80}},
					From: []networkingv1.NetworkPolicyPeer{
						{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"role": "frontend"}}},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{{Protocol: &proto, Port: &port80}},
				},
			},
		},
	}

	result, ok := BuildNetworkPolicyNode(policy)
	if !ok {
		t.Fatal("expected ok=true")
	}
	props := result.Node.Properties

	if got := props["policyTypes"].([]string); len(got) != 2 {
		t.Fatalf("policyTypes = %v, want 2 entries", got)
	}
	if got := props["podSelector"].([]string); !strContains(got, "app=web") {
		t.Fatalf("podSelector = %v, want to contain 'app=web'", got)
	}
	if got := props["ingress"].([]string); len(got) != 1 {
		t.Fatalf("ingress = %v, want 1 rule", got)
	}
	if got := props["egress"].([]string); len(got) != 1 {
		t.Fatalf("egress = %v, want 1 rule", got)
	}
}

func TestBuildNetworkPolicyNode_AllowAllIngress(t *testing.T) {
	// An empty ingress rule (no ports, no from) means allow all ingress.
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-all", Namespace: "ns1"},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			PodSelector: metav1.LabelSelector{},
			Ingress:     []networkingv1.NetworkPolicyIngressRule{{}},
		},
	}

	result, ok := BuildNetworkPolicyNode(policy)
	if !ok {
		t.Fatal("expected ok=true")
	}
	ingress := result.Node.Properties["ingress"].([]string)
	if len(ingress) != 1 || ingress[0] != "allow all" {
		t.Fatalf("ingress = %v, want [\"allow all\"]", ingress)
	}
}

func TestBuildNetworkPolicyNode_DenyAllIngress(t *testing.T) {
	// PolicyTypes includes Ingress but no ingress rules → deny all.
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "deny-all", Namespace: "ns1"},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			PodSelector: metav1.LabelSelector{},
			Ingress:     nil,
		},
	}

	result, ok := BuildNetworkPolicyNode(policy)
	if !ok {
		t.Fatal("expected ok=true")
	}
	ingress := result.Node.Properties["ingress"].([]string)
	if len(ingress) != 1 || ingress[0] != "<deny all>" {
		t.Fatalf("ingress = %v, want [\"<deny all>\"]", ingress)
	}
}

func TestBuildNetworkPolicyNode_PodSelectorEmpty(t *testing.T) {
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns1"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
		},
	}

	result, ok := BuildNetworkPolicyNode(policy)
	if !ok {
		t.Fatal("expected ok=true")
	}
	sel := result.Node.Properties["podSelector"].([]string)
	if len(sel) != 1 || sel[0] != "<all pods>" {
		t.Fatalf("podSelector = %v, want [\"<all pods>\"]", sel)
	}
}

func TestBuildNetworkPolicyNode_PodSelectorMatchLabels(t *testing.T) {
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns1"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "web", "env": "prod"},
			},
		},
	}

	result, ok := BuildNetworkPolicyNode(policy)
	if !ok {
		t.Fatal("expected ok=true")
	}
	sel := result.Node.Properties["podSelector"].([]string)
	if !strContains(sel, "app=web") || !strContains(sel, "env=prod") {
		t.Fatalf("podSelector = %v, want app=web and env=prod", sel)
	}
}

func TestBuildNetworkPolicyNode_IPBlockPeerWithExcept(t *testing.T) {
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns1"},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			PodSelector: metav1.LabelSelector{},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{IPBlock: &networkingv1.IPBlock{
							CIDR:   "10.0.0.0/8",
							Except: []string{"10.0.1.0/24"},
						}},
					},
				},
			},
		},
	}

	result, ok := BuildNetworkPolicyNode(policy)
	if !ok {
		t.Fatal("expected ok=true")
	}
	ingress := result.Node.Properties["ingress"].([]string)
	if len(ingress) == 0 {
		t.Fatal("expected ingress rules")
	}
	rule := ingress[0]
	if !contains(rule, "cidr 10.0.0.0/8") || !contains(rule, "10.0.1.0/24") {
		t.Fatalf("ingress rule = %q, want cidr + except", rule)
	}
}

func TestBuildNetworkPolicyNode_NamespacedAndPodSelectorPeer(t *testing.T) {
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns1"},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			PodSelector: metav1.LabelSelector{},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
							NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"env": "prod"}},
						},
					},
				},
			},
		},
	}

	result, ok := BuildNetworkPolicyNode(policy)
	if !ok {
		t.Fatal("expected ok=true")
	}
	ingress := result.Node.Properties["ingress"].([]string)
	if len(ingress) == 0 {
		t.Fatal("expected ingress rules")
	}
	rule := ingress[0]
	if !contains(rule, "app=web") || !contains(rule, "env=prod") || !contains(rule, "in namespaces") {
		t.Fatalf("ingress rule = %q, want combined pod+namespace selector", rule)
	}
}

func TestBuildNetworkPolicyNode_PortEndPortRange(t *testing.T) {
	proto := corev1.ProtocolTCP
	startPort := intstr.FromInt(8080)
	endPort := int32(8090)
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns1"},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			PodSelector: metav1.LabelSelector{},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: &proto, Port: &startPort, EndPort: &endPort},
					},
				},
			},
		},
	}

	result, ok := BuildNetworkPolicyNode(policy)
	if !ok {
		t.Fatal("expected ok=true")
	}
	ingress := result.Node.Properties["ingress"].([]string)
	if len(ingress) == 0 {
		t.Fatal("expected ingress rules")
	}
	rule := ingress[0]
	if !contains(rule, "8080-8090") {
		t.Fatalf("ingress rule = %q, want port range 8080-8090", rule)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
