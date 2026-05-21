package workload

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestBuildEnvDefinitionsFromContainers(t *testing.T) {
	containers := []corev1.Container{
		{
			Name: "app",
			Env: []corev1.EnvVar{
				{Name: "PLAIN", Value: "abc"},
				{Name: "S", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "sec"}, Key: "token"}}},
				{Name: "C", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "cm"}, Key: "flag"}}},
			},
			EnvFrom: []corev1.EnvFromSource{
				{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "sec-all"}}},
				{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "cm-all"}}},
			},
		},
	}

	defs := buildEnvDefinitionsFromContainers(containers, false, "Deployment", "web")
	if len(defs) != 5 {
		t.Fatalf("expected 5 env definitions, got %d", len(defs))
	}
	if defs[0].ValueSourceType != "literal" || defs[0].Value != "abc" {
		t.Fatalf("unexpected first definition: %#v", defs[0])
	}
	if defs[1].ValueSourceType != "secretKeyRef" || defs[1].RefName != "sec" || defs[1].RefKey != "token" {
		t.Fatalf("unexpected secret definition: %#v", defs[1])
	}
	if defs[4].ValueSourceType != "envFromConfigMapRef" || defs[4].RefName != "cm-all" {
		t.Fatalf("unexpected envFrom configmap definition: %#v", defs[4])
	}
}
