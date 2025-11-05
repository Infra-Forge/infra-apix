package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// EncodeDocument serialises the OpenAPI document in the requested format and returns payload and content type.
func EncodeDocument(doc *openapi3.T, format string) ([]byte, string, error) {
	switch strings.ToLower(format) {
	case "yaml", "yml", "":
		data, err := yaml.Marshal(doc)
		return data, "application/yaml", err
	case "json":
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(doc); err != nil {
			return nil, "", fmt.Errorf("encode json: %w", err)
		}
		return buf.Bytes(), "application/json", nil
	default:
		return nil, "", fmt.Errorf("unsupported format %q", format)
	}
}
