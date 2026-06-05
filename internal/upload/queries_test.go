package upload

import (
	"strings"
	"testing"
)

func makeQueries(queryStr string) []map[string]any {
	return []map[string]any{
		{"name": "Test Query", "description": "desc", "query": queryStr},
	}
}

func TestRenderQueryTemplatesNamedCluster(t *testing.T) {
	queries := makeQueries("WHERE s.cluster = '{{.Cluster}}'")
	if err := renderQueryTemplates(queries, "prod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := queries[0]["query"].(string)
	if got != "WHERE s.cluster = 'prod'" {
		t.Fatalf("want 'prod', got %q", got)
	}
}

func TestRenderQueryTemplatesExplicitDefault(t *testing.T) {
	queries := makeQueries("WHERE s.cluster = '{{.Cluster}}'")
	if err := renderQueryTemplates(queries, "default"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := queries[0]["query"].(string)
	if got != "WHERE s.cluster = 'default'" {
		t.Fatalf("want 'default', got %q", got)
	}
}

func TestRenderQueryTemplatesEmptyStringNormalisesToDefault(t *testing.T) {
	queries := makeQueries("WHERE s.cluster = '{{.Cluster}}'")
	if err := renderQueryTemplates(queries, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := queries[0]["query"].(string)
	if got != "WHERE s.cluster = 'default'" {
		t.Fatalf("want 'default' for empty input, got %q", got)
	}
}

func TestRenderQueryTemplatesWhitespaceOnlyNormalisesToDefault(t *testing.T) {
	queries := makeQueries("WHERE s.cluster = '{{.Cluster}}'")
	if err := renderQueryTemplates(queries, "   "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := queries[0]["query"].(string)
	if got != "WHERE s.cluster = 'default'" {
		t.Fatalf("want 'default' for whitespace-only input, got %q", got)
	}
}

func TestRenderQueryTemplatesNonQueryFieldsUntouched(t *testing.T) {
	queries := []map[string]any{
		{
			"name":        "unchanged name",
			"description": "unchanged description",
			"query":       "WHERE s.cluster = '{{.Cluster}}'",
		},
	}
	if err := renderQueryTemplates(queries, "prod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if queries[0]["name"] != "unchanged name" {
		t.Fatalf("name field was modified: %v", queries[0]["name"])
	}
	if queries[0]["description"] != "unchanged description" {
		t.Fatalf("description field was modified: %v", queries[0]["description"])
	}
}

func TestRenderQueryTemplatesNoToken(t *testing.T) {
	const raw = "WHERE s.kind = 'Pod'"
	queries := makeQueries(raw)
	if err := renderQueryTemplates(queries, "prod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := queries[0]["query"].(string)
	if got != raw {
		t.Fatalf("query without token should be unchanged, got %q", got)
	}
}

func TestRenderQueryTemplatesInvalidTemplate(t *testing.T) {
	queries := makeQueries("WHERE s.cluster = '{{.Unclosed'")
	err := renderQueryTemplates(queries, "prod")
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
	if !strings.Contains(err.Error(), "query 1") {
		t.Fatalf("expected error to mention query index, got: %v", err)
	}
}
