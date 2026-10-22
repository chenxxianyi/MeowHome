package openapi

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRecordRoutesAreDocumented(t *testing.T) {
	data, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths      map[string]map[string]map[string]any `yaml:"paths"`
		Components struct {
			Schemas map[string]any `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse OpenAPI document: %v", err)
	}
	for path, methods := range map[string][]string{
		"/families/{familyId}/records":                               {"get", "post"},
		"/families/{familyId}/records/{recordId}":                    {"get", "patch", "delete"},
		"/families/{familyId}/records/{recordId}/vaccine-suggestion": {"get"},
		"/families/{familyId}/records/{recordId}/vaccine-reminders":  {"post"},
	} {
		for _, method := range methods {
			if _, ok := document.Paths[path][method]; !ok {
				t.Errorf("OpenAPI is missing %s %s", method, path)
			}
		}
	}
	for _, schema := range []string{"Record", "RecordInput", "RecordUpdateRequest"} {
		if _, ok := document.Components.Schemas[schema]; !ok {
			t.Errorf("OpenAPI is missing %s schema", schema)
		}
	}
}
