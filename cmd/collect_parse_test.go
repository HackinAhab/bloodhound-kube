package cmd

import "testing"

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
