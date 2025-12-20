package plugins

import (
	apix "github.com/Infra-Forge/infra-apix"
	"github.com/getkin/kin-openapi/openapi3"
)

// CustomServersPlugin adds custom server URLs to the OpenAPI spec.
// This plugin demonstrates the OnSpecBuild hook.
//
// Example usage:
//
//	plugin := &plugins.CustomServersPlugin{
//		Servers: []*openapi3.Server{
//			{URL: "https://api.example.com", Description: "Production"},
//			{URL: "https://staging.example.com", Description: "Staging"},
//			{URL: "http://localhost:8080", Description: "Development"},
//		},
//	}
//	apix.RegisterPlugin(plugin)
type CustomServersPlugin struct {
	apix.BasePlugin
	// Servers to add to the OpenAPI spec
	Servers []*openapi3.Server
}

// NewCustomServersPlugin creates a new CustomServersPlugin.
func NewCustomServersPlugin(servers []*openapi3.Server) *CustomServersPlugin {
	return &CustomServersPlugin{
		BasePlugin: apix.BasePlugin{PluginName: "custom-servers"},
		Servers:    servers,
	}
}

// OnSpecBuild adds custom servers to the OpenAPI document.
func (p *CustomServersPlugin) OnSpecBuild(doc *openapi3.T) error {
	doc.Servers = append(doc.Servers, p.Servers...)
	return nil
}
