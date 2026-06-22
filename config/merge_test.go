package config

import (
	"encoding/json"
	"testing"
)

func TestMergeQueries(t *testing.T) {
	tests := []struct {
		name          string
		embedded      string
		user          string
		expectedCount int
		expectedNames []string
		expectError   bool
	}{
		{
			name:          "no overlap",
			embedded:      `{"queries":[{"name":"Q1","description":"D1","query":"MATCH 1"}]}`,
			user:          `{"queries":[{"name":"Q2","description":"D2","query":"MATCH 2"}]}`,
			expectedCount: 2,
			expectedNames: []string{"Q2", "Q1"},
		},
		{
			name:          "user overrides embedded",
			embedded:      `{"queries":[{"name":"Q1","description":"D1","query":"MATCH 1"}]}`,
			user:          `{"queries":[{"name":"Q1","description":"D2","query":"MATCH 2"}]}`,
			expectedCount: 1,
			expectedNames: []string{"Q1"},
		},
		{
			name:          "empty user",
			embedded:      `{"queries":[{"name":"Q1","description":"D1","query":"MATCH 1"}]}`,
			user:          `{"queries":[]}`,
			expectedCount: 1,
			expectedNames: []string{"Q1"},
		},
		{
			name:          "empty embedded",
			embedded:      `{"queries":[]}`,
			user:          `{"queries":[{"name":"Q1","description":"D1","query":"MATCH 1"}]}`,
			expectedCount: 1,
			expectedNames: []string{"Q1"},
		},
		{
			name:          "multiple queries with partial overlap",
			embedded:      `{"queries":[{"name":"Q1","description":"D1","query":"MATCH 1"},{"name":"Q2","description":"D2","query":"MATCH 2"}]}`,
			user:          `{"queries":[{"name":"Q2","description":"D2-override","query":"MATCH 2-override"},{"name":"Q3","description":"D3","query":"MATCH 3"}]}`,
			expectedCount: 3,
			expectedNames: []string{"Q2", "Q3", "Q1"},
		},
		{
			name:        "invalid embedded JSON",
			embedded:    `{invalid json}`,
			user:        `{"queries":[]}`,
			expectError: true,
		},
		{
			name:        "invalid user JSON",
			embedded:    `{"queries":[]}`,
			user:        `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeQueries([]byte(tt.embedded), []byte(tt.user))
			
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("MergeQueries failed: %v", err)
			}

			var config QueriesConfig
			if err := json.Unmarshal(result, &config); err != nil {
				t.Fatalf("Unmarshal result failed: %v", err)
			}

			if len(config.Queries) != tt.expectedCount {
				t.Errorf("Expected %d queries, got %d", tt.expectedCount, len(config.Queries))
			}

			for i, name := range tt.expectedNames {
				if i >= len(config.Queries) {
					t.Errorf("Expected query %d to be %s, but only got %d queries", i, name, len(config.Queries))
					continue
				}
				if config.Queries[i].Name != name {
					t.Errorf("Expected query %d to be %s, got %s", i, name, config.Queries[i].Name)
				}
			}
		})
	}
}

func TestMergeSchema(t *testing.T) {
	tests := []struct {
		name          string
		embedded      string
		user          string
		expectedCount int
		expectedNames []string
		expectError   bool
	}{
		{
			name:          "no overlap",
			embedded:      `{"node_kinds":[{"name":"BHK_Type1","icon":"icon1","color":"#000000"}],"relationship_kinds":[]}`,
			user:          `{"node_kinds":[{"name":"BHK_Type2","icon":"icon2","color":"#FFFFFF"}],"relationship_kinds":[]}`,
			expectedCount: 2,
			expectedNames: []string{"BHK_Type1", "BHK_Type2"},
		},
		{
			name:          "user overrides embedded",
			embedded:      `{"node_kinds":[{"name":"BHK_Type1","icon":"icon1","color":"#000000"}],"relationship_kinds":[]}`,
			user:          `{"node_kinds":[{"name":"BHK_Type1","icon":"icon1-override","color":"#FFFFFF"}],"relationship_kinds":[]}`,
			expectedCount: 1,
			expectedNames: []string{"BHK_Type1"},
		},
		{
			name:          "empty user",
			embedded:      `{"node_kinds":[{"name":"BHK_Type1","icon":"icon1","color":"#000000"}],"relationship_kinds":[]}`,
			user:          `{"node_kinds":[],"relationship_kinds":[]}`,
			expectedCount: 1,
			expectedNames: []string{"BHK_Type1"},
		},
		{
			name:          "empty embedded",
			embedded:      `{"node_kinds":[],"relationship_kinds":[]}`,
			user:          `{"node_kinds":[{"name":"BHK_Type1","icon":"icon1","color":"#000000"}],"relationship_kinds":[]}`,
			expectedCount: 1,
			expectedNames: []string{"BHK_Type1"},
		},
		{
			name:          "multiple types with partial overlap",
			embedded:      `{"node_kinds":[{"name":"BHK_Type1","icon":"icon1","color":"#000000"},{"name":"BHK_Type2","icon":"icon2","color":"#111111"}],"relationship_kinds":[]}`,
			user:          `{"node_kinds":[{"name":"BHK_Type2","icon":"icon2-override","color":"#FFFFFF"},{"name":"BHK_Type3","icon":"icon3","color":"#222222"}],"relationship_kinds":[]}`,
			expectedCount: 3,
			expectedNames: []string{"BHK_Type1", "BHK_Type2", "BHK_Type3"},
		},
		{
			name:        "invalid embedded JSON",
			embedded:    `{invalid json}`,
			user:        `{"node_kinds":[],"relationship_kinds":[]}`,
			expectError: true,
		},
		{
			name:        "invalid user JSON",
			embedded:    `{"node_kinds":[],"relationship_kinds":[]}`,
			user:        `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeSchema([]byte(tt.embedded), []byte(tt.user))

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("MergeSchema failed: %v", err)
			}

			var config CustomTypesConfig
			if err := json.Unmarshal(result, &config); err != nil {
				t.Fatalf("Unmarshal result failed: %v", err)
			}

			if len(config.NodeKinds) != tt.expectedCount {
				t.Errorf("Expected %d node kinds, got %d", tt.expectedCount, len(config.NodeKinds))
			}

			nameSet := map[string]bool{}
			for _, nk := range config.NodeKinds {
				nameSet[nk.Name] = true
			}
			for _, name := range tt.expectedNames {
				if !nameSet[name] {
					t.Errorf("Expected node kind %s to exist, but it doesn't", name)
				}
			}
		})
	}
}

func TestMergeQueriesUserPrecedence(t *testing.T) {
	embedded := `{"queries":[{"name":"Test","description":"Embedded","query":"MATCH (e)"}]}`
	user := `{"queries":[{"name":"Test","description":"User","query":"MATCH (u)"}]}`

	result, err := MergeQueries([]byte(embedded), []byte(user))
	if err != nil {
		t.Fatalf("MergeQueries failed: %v", err)
	}

	var config QueriesConfig
	if err := json.Unmarshal(result, &config); err != nil {
		t.Fatalf("Unmarshal result failed: %v", err)
	}

	if len(config.Queries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(config.Queries))
	}

	if config.Queries[0].Description != "User" {
		t.Errorf("Expected user query to take precedence, got description: %s", config.Queries[0].Description)
	}

	if config.Queries[0].Query != "MATCH (u)" {
		t.Errorf("Expected user query to take precedence, got query: %s", config.Queries[0].Query)
	}
}

func TestMergeSchemaUserPrecedence(t *testing.T) {
	embedded := `{"node_kinds":[{"name":"BHK_Test","icon":"embedded","color":"#000000"}],"relationship_kinds":[]}`
	user := `{"node_kinds":[{"name":"BHK_Test","icon":"user","color":"#FFFFFF"}],"relationship_kinds":[]}`

	result, err := MergeSchema([]byte(embedded), []byte(user))
	if err != nil {
		t.Fatalf("MergeSchema failed: %v", err)
	}

	var config CustomTypesConfig
	if err := json.Unmarshal(result, &config); err != nil {
		t.Fatalf("Unmarshal result failed: %v", err)
	}

	if len(config.NodeKinds) != 1 {
		t.Fatalf("Expected 1 node kind, got %d", len(config.NodeKinds))
	}

	if config.NodeKinds[0].Icon != "user" {
		t.Errorf("Expected user type to take precedence, got icon: %s", config.NodeKinds[0].Icon)
	}

	if config.NodeKinds[0].Color != "#FFFFFF" {
		t.Errorf("Expected user type to take precedence, got color: %s", config.NodeKinds[0].Color)
	}
}
