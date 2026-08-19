package route

import (
	"slices"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildRouteNode_BasicProperties(t *testing.T) {
	rt := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{Name: "my-route", Namespace: "default"},
		Spec: routev1.RouteSpec{
			Host: "my-route.apps.example.com",
			To:   routev1.RouteTargetReference{Kind: "Service", Name: "my-svc"},
			TLS:  &routev1.TLSConfig{Termination: routev1.TLSTerminationEdge},
		},
	}

	result, ok := BuildRouteNode(rt)
	if !ok {
		t.Fatal("expected ok=true")
	}

	if result.Node.ID != "BHK_Route:default:my-route" {
		t.Fatalf("ID = %q, want BHK_Route:default:my-route", result.Node.ID)
	}
	if !slices.Contains(result.Node.Kinds, "BHK_Route") {
		t.Fatalf("Kinds = %v, want to contain BHK_Route", result.Node.Kinds)
	}

	props := result.Node.Properties
	if got := props["urls"].([]string); len(got) != 1 || got[0] != "https://my-route.apps.example.com/" {
		t.Fatalf("urls = %v, want [https://my-route.apps.example.com/]", got)
	}
	if got := props["backendRefKeys"].([]string); len(got) != 1 || got[0] != "default/my-svc" {
		t.Fatalf("backendRefKeys = %v, want [default/my-svc]", got)
	}

	if len(result.Core) != 1 {
		t.Fatalf("Core = %v, want 1 entry", result.Core)
	}
	core := result.Core[0].Data.(Route)
	if len(core.BackendRefs) != 1 || core.BackendRefs[0].Name != "my-svc" || core.BackendRefs[0].Namespace != "default" {
		t.Fatalf("BackendRefs = %v, want [{default my-svc}]", core.BackendRefs)
	}
}

func TestBuildRouteNode_NoTLS_HTTP(t *testing.T) {
	rt := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{Name: "plain", Namespace: "ns1"},
		Spec: routev1.RouteSpec{
			Host: "plain.apps.example.com",
			Path: "/api",
			To:   routev1.RouteTargetReference{Kind: "Service", Name: "svc1"},
		},
	}

	result, ok := BuildRouteNode(rt)
	if !ok {
		t.Fatal("expected ok=true")
	}
	urls := result.Node.Properties["urls"].([]string)
	if len(urls) != 1 || urls[0] != "http://plain.apps.example.com/api" {
		t.Fatalf("urls = %v, want [http://plain.apps.example.com/api]", urls)
	}
}

func TestBuildRouteNode_AlternateBackends(t *testing.T) {
	rt := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{Name: "canary", Namespace: "ns1"},
		Spec: routev1.RouteSpec{
			Host: "canary.apps.example.com",
			To:   routev1.RouteTargetReference{Kind: "Service", Name: "svc-main"},
			AlternateBackends: []routev1.RouteTargetReference{
				{Kind: "Service", Name: "svc-canary"},
			},
		},
	}

	result, ok := BuildRouteNode(rt)
	if !ok {
		t.Fatal("expected ok=true")
	}
	keys := result.Node.Properties["backendRefKeys"].([]string)
	if len(keys) != 2 || !slices.Contains(keys, "ns1/svc-main") || !slices.Contains(keys, "ns1/svc-canary") {
		t.Fatalf("backendRefKeys = %v, want ns1/svc-main and ns1/svc-canary", keys)
	}
}

func TestBuildRouteNode_NoHost(t *testing.T) {
	rt := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{Name: "no-host", Namespace: "ns1"},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{Kind: "Service", Name: "svc1"},
		},
	}

	result, ok := BuildRouteNode(rt)
	if !ok {
		t.Fatal("expected ok=true")
	}
	urls := result.Node.Properties["urls"].([]string)
	if len(urls) != 0 {
		t.Fatalf("urls = %v, want empty", urls)
	}
}

func TestBuildRouteNode_NoName(t *testing.T) {
	rt := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns1"},
	}
	if _, ok := BuildRouteNode(rt); ok {
		t.Fatal("expected ok=false for missing name")
	}
}

func TestBuildRouteNode_WrongType(t *testing.T) {
	if _, ok := BuildRouteNode(nil); ok {
		t.Fatal("expected ok=false for nil object")
	}
	if _, ok := BuildRouteNode(&corev1.Pod{}); ok {
		t.Fatal("expected ok=false for non-Route object")
	}
}
