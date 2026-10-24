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

// TestShoppingAndPurchaseRoutesAreDocumented 校验待购清单与采购导入契约
// 与后端路由同语义，避免前后端 DTO 漂移。
func TestShoppingAndPurchaseRoutesAreDocumented(t *testing.T) {
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
		"/families/{familyId}/shopping-list":                             {"get", "post"},
		"/families/{familyId}/shopping-list/{itemId}":                    {"patch", "delete"},
		"/families/{familyId}/inventory/{itemId}/adjustments":            {"post"},
		"/families/{familyId}/inventory/transactions":                    {"get"},
		"/families/{familyId}/purchase-media":                            {"post"},
		"/families/{familyId}/purchase-imports":                          {"post", "get"},
		"/families/{familyId}/purchase-imports/{importId}":               {"get", "patch"},
		"/families/{familyId}/purchase-imports/{importId}/recognize":     {"post"},
		"/families/{familyId}/purchase-imports/{importId}/confirm":       {"post"},
		"/families/{familyId}/purchase-imports/{importId}/cancel":        {"post"},
		"/families/{familyId}/purchase-imports/{importId}/media/{index}": {"get", "delete"},
	} {
		for _, method := range methods {
			if _, ok := document.Paths[path][method]; !ok {
				t.Errorf("OpenAPI is missing %s %s", method, path)
			}
		}
	}
	for _, schema := range []string{
		"ShoppingItem", "ShoppingInput", "ShoppingUpdateInput",
		"InventoryAdjustRequest", "InventoryTransaction",
		"PurchaseImport", "PurchaseLine", "PurchaseDraftRequest",
		"PurchaseDraftLineInput", "PurchaseConfirmRequest", "PurchaseConfirmSummary", "PurchaseRecognizeResult",
	} {
		if _, ok := document.Components.Schemas[schema]; !ok {
			t.Errorf("OpenAPI is missing %s schema", schema)
		}
	}
}
