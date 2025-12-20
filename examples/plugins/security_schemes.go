package plugins

import (
	apix "github.com/Infra-Forge/infra-apix"
	"github.com/getkin/kin-openapi/openapi3"
)

// SecuritySchemesPlugin adds security schemes to the OpenAPI spec.
// This plugin demonstrates the OnSpecBuild hook for adding authentication.
//
// Example usage:
//
//	plugin := &plugins.SecuritySchemesPlugin{
//		Schemes: map[string]*openapi3.SecurityScheme{
//			"bearerAuth": {
//				Type:         "http",
//				Scheme:       "bearer",
//				BearerFormat: "JWT",
//			},
//			"apiKey": {
//				Type: "apiKey",
//				In:   "header",
//				Name: "X-API-Key",
//			},
//		},
//		GlobalSecurity: []map[string][]string{
//			{"bearerAuth": {}},
//		},
//	}
//	apix.RegisterPlugin(plugin)
type SecuritySchemesPlugin struct {
	apix.BasePlugin
	// Schemes to add to the OpenAPI spec
	Schemes map[string]*openapi3.SecurityScheme
	// GlobalSecurity applies security requirements globally
	GlobalSecurity openapi3.SecurityRequirements
}

// NewSecuritySchemesPlugin creates a new SecuritySchemesPlugin.
func NewSecuritySchemesPlugin(schemes map[string]*openapi3.SecurityScheme, globalSecurity openapi3.SecurityRequirements) *SecuritySchemesPlugin {
	return &SecuritySchemesPlugin{
		BasePlugin:     apix.BasePlugin{PluginName: "security-schemes"},
		Schemes:        schemes,
		GlobalSecurity: globalSecurity,
	}
}

// OnSpecBuild adds security schemes to the OpenAPI document.
func (p *SecuritySchemesPlugin) OnSpecBuild(doc *openapi3.T) error {
	if doc.Components == nil {
		doc.Components = &openapi3.Components{}
	}
	if doc.Components.SecuritySchemes == nil {
		doc.Components.SecuritySchemes = make(map[string]*openapi3.SecuritySchemeRef)
	}

	// Add security schemes
	for name, scheme := range p.Schemes {
		doc.Components.SecuritySchemes[name] = &openapi3.SecuritySchemeRef{
			Value: scheme,
		}
	}

	// Add global security requirements
	if len(p.GlobalSecurity) > 0 {
		doc.Security = append(doc.Security, p.GlobalSecurity...)
	}

	return nil
}
