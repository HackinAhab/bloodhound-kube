package collector

import "testing"

func TestApplyCollectionHelpers_RedactsPodEnvValues(t *testing.T) {
	obj := map[string]any{
		"spec": map[string]any{
			"containers": []any{
				map[string]any{
					"name": "app",
					"env": []any{
						map[string]any{"name": "A", "value": "plain"},
					},
				},
			},
		},
	}

	updated := applyCollectionHelpers(obj, "pods", true)
	if updated == nil {
		t.Fatalf("expected object")
	}

	spec := updated["spec"].(map[string]any)
	containers := spec["containers"].([]any)
	container := containers[0].(map[string]any)
	env := container["env"].([]any)
	entry := env[0].(map[string]any)
	if value := entry["value"]; value != "" {
		t.Fatalf("expected env value to be redacted, got %#v", value)
	}
}

func TestApplyCollectionHelpers_DoesNotRedactWhenDisabled(t *testing.T) {
	obj := map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"env": []any{map[string]any{"name": "A", "value": "plain"}},
						},
					},
				},
			},
		},
	}

	updated := applyCollectionHelpers(obj, "deployments", false)
	spec := updated["spec"].(map[string]any)
	template := spec["template"].(map[string]any)
	podSpec := template["spec"].(map[string]any)
	containers := podSpec["containers"].([]any)
	container := containers[0].(map[string]any)
	env := container["env"].([]any)
	entry := env[0].(map[string]any)
	if value := entry["value"]; value != "plain" {
		t.Fatalf("expected env value to stay plain, got %#v", value)
	}
}
