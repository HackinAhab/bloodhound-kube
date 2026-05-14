package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bloodhound-kube/internal/utils"
)

func TestDeriveParsedOutputPath(t *testing.T) {
	got := deriveParsedOutputPath("/tmp/collection.jsonl")
	if got != "/tmp/collection.json" {
		t.Fatalf("expected /tmp/collection.json, got %q", got)
	}

	got = deriveParsedOutputPath("collection")
	if got != "collection.json" {
		t.Fatalf("expected collection.json, got %q", got)
	}
}

func TestResolveParsedOutputPath(t *testing.T) {
	jsonlPath := "/tmp/run/collection.jsonl"

	if got := resolveParsedOutputPath(jsonlPath, ""); got != "/tmp/run/collection.json" {
		t.Fatalf("unexpected default parsed output path: %q", got)
	}

	if got := resolveParsedOutputPath(jsonlPath, "/out/"); got != "/out/collection.json" {
		t.Fatalf("unexpected directory parsed output path: %q", got)
	}

	if got := resolveParsedOutputPath(jsonlPath, "/out/custom.json"); got != "/out/custom.json" {
		t.Fatalf("unexpected explicit parsed output path: %q", got)
	}
}

func TestParseServiceRunValidation(t *testing.T) {
	service := ParseService{}
	log := utils.New("error", true)

	_, err := service.Run(ParseRequest{}, log)
	if err == nil || !strings.Contains(err.Error(), "input file is required") {
		t.Fatalf("expected input file validation error, got %v", err)
	}
}

func TestParseServiceRunWritesOutput(t *testing.T) {
	service := ParseService{}
	log := utils.New("error", true)

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.jsonl")
	outputPath := filepath.Join(tmpDir, "output.json")

	jsonl := `{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"team-a"}}` + "\n" +
		`{"apiVersion":"v1","kind":"ServiceAccount","metadata":{"name":"reader","namespace":"team-a"}}` + "\n" +
		`{"apiVersion":"rbac.authorization.k8s.io/v1","kind":"Role","metadata":{"name":"secret-reader","namespace":"team-a"},"rules":[{"apiGroups":[""],"resources":["secrets"],"verbs":["get"]}]}` + "\n" +
		`{"apiVersion":"rbac.authorization.k8s.io/v1","kind":"RoleBinding","metadata":{"name":"bind-secret-reader","namespace":"team-a"},"roleRef":{"apiGroup":"rbac.authorization.k8s.io","kind":"Role","name":"secret-reader"},"subjects":[{"kind":"ServiceAccount","name":"reader","namespace":"team-a"}]}` + "\n"

	if err := os.WriteFile(inputPath, []byte(jsonl), 0644); err != nil {
		t.Fatalf("failed to write input: %v", err)
	}

	resp, err := service.Run(ParseRequest{InputPath: inputPath, OutputPath: outputPath, ClusterName: "prod"}, log)
	if err != nil {
		t.Fatalf("parse run failed: %v", err)
	}
	if resp.NodeCount == 0 {
		t.Fatalf("expected nodes to be created")
	}

	out, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, "prod:") {
		t.Fatalf("expected cluster-scoped IDs in output")
	}
	if !strings.Contains(output, "ServiceAccount") {
		t.Fatalf("expected ServiceAccount node in output")
	}
}
