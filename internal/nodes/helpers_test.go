package nodes

import (
	"reflect"
	"testing"
)

func TestBuildID(t *testing.T) {
	if got := BuildID("Pod", "default", "nginx"); got != "Pod:default:nginx" {
		t.Fatalf("unexpected BuildID: %s", got)
	}
	if got := BuildID("Node", "", "worker-1"); got != "Node:worker-1" {
		t.Fatalf("unexpected BuildID without namespace: %s", got)
	}
}

func TestGetters(t *testing.T) {
	input := map[string]any{
		"meta":   map[string]any{"name": "kube"},
		"items":  []any{"a", "b"},
		"name":   "value",
		"ready":  true,
		"number": float64(3),
	}

	if got := GetString(GetMap(input, "meta"), "name"); got != "kube" {
		t.Fatalf("expected kube, got %s", got)
	}
	if got := GetSlice(input, "items"); len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got := GetString(input, "name"); got != "value" {
		t.Fatalf("expected value, got %s", got)
	}
	if got := GetStringDefault(input, "missing", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %s", got)
	}
	if got := GetBool(input, "ready"); got != true {
		t.Fatalf("expected true, got %v", got)
	}
	if got := GetNumber(input, "number"); got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
}

func TestMapHelpers(t *testing.T) {
	input := map[string]any{"b": 2, "a": 1}
	wantList := []string{"a=1", "b=2"}
	if got := MapToSortedList(input); !reflect.DeepEqual(got, wantList) {
		t.Fatalf("expected %v, got %v", wantList, got)
	}

	wantKeys := []string{"a", "b"}
	if got := MapKeysSorted(input); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("expected %v, got %v", wantKeys, got)
	}
}

func TestSortedSetKeys(t *testing.T) {
	set := map[string]struct{}{"b": {}, "a": {}}
	want := []string{"a", "b"}
	if got := SortedSetKeys(set); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestStringSlice(t *testing.T) {
	values := []any{"a", 1, "b"}
	want := []string{"a", "b"}
	if got := StringSlice(values); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestSeLinuxSummary(t *testing.T) {
	input := map[string]any{"user": "u", "role": "r", "type": "t", "level": "l"}
	want := "user=u, role=r, type=t, level=l"
	if got := SeLinuxSummary(input); got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestAppArmorProfileValue(t *testing.T) {
	if got := AppArmorProfileValue(map[string]any{"appArmorProfile": "RuntimeDefault"}); got != "RuntimeDefault" {
		t.Fatalf("unexpected string profile: %s", got)
	}

	if got := AppArmorProfileValue(map[string]any{"appArmorProfile": map[string]any{"type": "Localhost"}}); got != "Localhost" {
		t.Fatalf("unexpected map profile: %s", got)
	}
}

func TestRemoveKeyFromSlice(t *testing.T) {
	input := []string{"a", "b", "a"}
	want := []string{"b"}
	if got := RemoveKeyFromSlice(input, "a"); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
