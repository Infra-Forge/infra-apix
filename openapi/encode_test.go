package openapi

import (
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestEncodeDocumentJSON(t *testing.T) {
	doc := &openapi3.T{OpenAPI: "3.1.0", Info: &openapi3.Info{Title: "API", Version: "1"}}
	data, ctype, err := EncodeDocument(doc, "json")
	if err != nil {
		t.Fatalf("encode json failed: %v", err)
	}
	if ctype != "application/json" {
		t.Fatalf("expected json content type, got %s", ctype)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if decoded["openapi"] != "3.1.0" {
		t.Fatalf("expected openapi version present")
	}
}

func TestEncodeDocumentYAML(t *testing.T) {
	doc := &openapi3.T{OpenAPI: "3.1.0", Info: &openapi3.Info{Title: "API", Version: "1"}}
	data, ctype, err := EncodeDocument(doc, "yaml")
	if err != nil {
		t.Fatalf("encode yaml failed: %v", err)
	}
	if ctype != "application/yaml" {
		t.Fatalf("expected yaml content type, got %s", ctype)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty payload")
	}
}
