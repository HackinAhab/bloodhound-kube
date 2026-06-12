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

func TestMergeCustomTypes(t *testing.T) {
	tests := []struct {
		name          string
		embedded      string
		user          string
		expectedCount int
		expectedTypes []string
		expectError   bool
	}{
		{
			name:          "no overlap",
			embedded:      `{"custom_types":{"Type1":{"icon":{"type":"font-awesome","name":"icon1","color":"#000000"}}}}`,
			user:          `{"custom_types":{"Type2":{"icon":{"type":"font-awesome","name":"icon2","color":"#FFFFFF"}}}}`,
			expectedCount: 2,
			expectedTypes: []string{"Type1", "Type2"},
		},
		{
			name:          "user overrides embedded",
			embedded:      `{"custom_types":{"Type1":{"icon":{"type":"font-awesome","name":"icon1","color":"#000000"}}}}`,
			user:          `{"custom_types":{"Type1":{"icon":{"type":"font-awesome","name":"icon1-override","color":"#FFFFFF"}}}}`,
			expectedCount: 1,
			expectedTypes: []string{"Type1"},
		},
		{
			name:          "empty user",
			embedded:      `{"custom_types":{"Type1":{"icon":{"type":"font-awesome","name":"icon1","color":"#000000"}}}}`,
			user:          `{"custom_types":{}}`,
			expectedCount: 1,
			expectedTypes: []string{"Type1"},
		},
		{
			name:          "empty embedded",
			embedded:      `{"custom_types":{}}`,
			user:          `{"custom_types":{"Type1":{"icon":{"type":"font-awesome","name":"icon1","color":"#000000"}}}}`,
			expectedCount: 1,
			expectedTypes: []string{"Type1"},
		},
		{
			name:          "multiple types with partial overlap",
			embedded:      `{"custom_types":{"Type1":{"icon":{"type":"font-awesome","name":"icon1","color":"#000000"}},"Type2":{"icon":{"type":"font-awesome","name":"icon2","color":"#111111"}}}}`,
			user:          `{"custom_types":{"Type2":{"icon":{"type":"font-awesome","name":"icon2-override","color":"#FFFFFF"}},"Type3":{"icon":{"type":"font-awesome","name":"icon3","color":"#222222"}}}}`,
			expectedCount: 3,
			expectedTypes: []string{"Type1", "Type2", "Type3"},
		},
		{
			name:        "invalid embedded JSON",
			embedded:    `{invalid json}`,
			user:        `{"custom_types":{}}`,
			expectError: true,
		},
		{
			name:        "invalid user JSON",
			embedded:    `{"custom_types":{}}`,
			user:        `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeCustomTypes([]byte(tt.embedded), []byte(tt.user))
			
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("MergeCustomTypes failed: %v", err)
			}

			var config CustomTypesConfig
			if err := json.Unmarshal(result, &config); err != nil {
				t.Fatalf("Unmarshal result failed: %v", err)
			}

			if len(config.CustomTypes) != tt.expectedCount {
				t.Errorf("Expected %d custom types, got %d", tt.expectedCount, len(config.CustomTypes))
			}

			for _, typeName := range tt.expectedTypes {
				if _, exists := config.CustomTypes[typeName]; !exists {
					t.Errorf("Expected custom type %s to exist, but it doesn't", typeName)
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

func TestMergeCustomTypesUserPrecedence(t *testing.T) {
	embedded := `{"custom_types":{"Test":{"icon":{"type":"font-awesome","name":"embedded","color":"#000000"}}}}`
	user := `{"custom_types":{"Test":{"icon":{"type":"font-awesome","name":"user","color":"#FFFFFF"}}}}`

	result, err := MergeCustomTypes([]byte(embedded), []byte(user))
	if err != nil {
		t.Fatalf("MergeCustomTypes failed: %v", err)
	}

	var config CustomTypesConfig
	if err := json.Unmarshal(result, &config); err != nil {
		t.Fatalf("Unmarshal result failed: %v", err)
	}

	if len(config.CustomTypes) != 1 {
		t.Fatalf("Expected 1 custom type, got %d", len(config.CustomTypes))
	}

	testType, exists := config.CustomTypes["Test"]
	if !exists {
		t.Fatalf("Expected 'Test' type to exist")
	}

	if testType.Icon.Name != "user" {
		t.Errorf("Expected user type to take precedence, got icon name: %s", testType.Icon.Name)
	}

	if testType.Icon.Color != "#FFFFFF" {
		t.Errorf("Expected user type to take precedence, got color: %s", testType.Icon.Color)
	}
}
