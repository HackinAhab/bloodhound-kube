package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bloodhound-kube/internal/utils"
)

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

func TestUploadServiceRunValidation(t *testing.T) {
	service := UploadService{}
	log := utils.New("error", true)

	err := service.Run(UploadRequest{}, log)
	if err == nil || !strings.Contains(err.Error(), "token ID and token key are required") {
		t.Fatalf("expected token validation error, got %v", err)
	}

	err = service.Run(UploadRequest{TokenID: "id", TokenKey: "key"}, log)
	if err == nil || !strings.Contains(err.Error(), "provide --model-file, --queries-file, --upload-file, or --reset") {
		t.Fatalf("expected action validation error, got %v", err)
	}
}
