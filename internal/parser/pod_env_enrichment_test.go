package parser

import (
	"testing"

	"bloodhound-kube/internal/nodes/workload"
)

func TestBuildEnvProperties(t *testing.T) {
	defs := []workload.EnvDefinition{
		{
			SourceKind:      "Pod",
			SourceName:      "pod-a",
			Container:       "web",
			InitContainer:   false,
			EnvName:         "LOG_LEVEL",
			Value:           "debug",
			ValueSourceType: "literal",
		},
		{
			SourceKind:      "Deployment",
			SourceName:      "web",
			Container:       "web",
			InitContainer:   false,
			EnvName:         "DB_PASSWORD",
			ValueSourceType: "secretKeyRef",
			RefName:         "db-creds",
			RefKey:          "password",
		},
	}

	props := buildEnvProperties(defs)
	if _, exists := props["env_count"]; exists {
		t.Fatalf("env_count should not be present")
	}

	entry, ok := props["Env00"].([]string)
	if !ok {
		t.Fatalf("expected Env00 to be []string, got %#v", props["Env00"])
	}
	if len(entry) == 0 {
		t.Fatalf("expected Env00 tokens")
	}
}

func TestBuildEnvProperties_DedupesPodAndControllerDuplicates(t *testing.T) {
	defs := []workload.EnvDefinition{
		{
			SourceKind:      "Deployment",
			SourceName:      "web",
			Container:       "web",
			EnvName:         "ACTUAL_PORT",
			Value:           "5006",
			ValueSourceType: "literal",
		},
		{
			SourceKind:      "Pod",
			SourceName:      "web-abc",
			Container:       "web",
			EnvName:         "ACTUAL_PORT",
			Value:           "5006",
			ValueSourceType: "literal",
		},
	}

	props := buildEnvProperties(defs)
	if _, exists := props["env_count"]; exists {
		t.Fatalf("env_count should not be present")
	}
	entry, ok := props["Env00"].([]string)
	if !ok {
		t.Fatalf("expected Env00 to be []string, got %#v", props["Env00"])
	}
	if len(entry) == 0 || entry[0] != "src=Deployment/web" {
		t.Fatalf("expected controller source to be retained, got %#v", entry)
	}
}
