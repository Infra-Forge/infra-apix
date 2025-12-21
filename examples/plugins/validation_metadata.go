package plugins

import (
	apix "github.com/Infra-Forge/infra-apix"
	"github.com/getkin/kin-openapi/openapi3"
)

// ValidationMetadataPlugin adds validation metadata to schemas.
// This plugin demonstrates the OnSchemaGenerate hook.
//
// Example usage:
//
//	plugin := &plugins.ValidationMetadataPlugin{
//		AddValidationExtension: true,
//		AddSchemaVersion:       "1.0",
//	}
//	apix.RegisterPlugin(plugin)
type ValidationMetadataPlugin struct {
	apix.BasePlugin
	// AddValidationExtension adds x-validated extension to all schemas
	AddValidationExtension bool
	// AddSchemaVersion adds x-schema-version extension to all schemas
	AddSchemaVersion string
}

// NewValidationMetadataPlugin creates a new ValidationMetadataPlugin.
func NewValidationMetadataPlugin(addValidation bool, schemaVersion string) *ValidationMetadataPlugin {
	return &ValidationMetadataPlugin{
		BasePlugin:             apix.BasePlugin{PluginName: "validation-metadata"},
		AddValidationExtension: addValidation,
		AddSchemaVersion:       schemaVersion,
	}
}

// OnSchemaGenerate adds validation metadata to schemas.
func (p *ValidationMetadataPlugin) OnSchemaGenerate(typeName string, schema *openapi3.Schema) error {
	if schema.Extensions == nil {
		schema.Extensions = make(map[string]interface{})
	}

	if p.AddValidationExtension {
		schema.Extensions["x-validated"] = true
	}

	if p.AddSchemaVersion != "" {
		schema.Extensions["x-schema-version"] = p.AddSchemaVersion
	}

	return nil
}
