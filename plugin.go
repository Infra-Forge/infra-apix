package apix

import (
	"github.com/Infra-Forge/infra-apix/internal/logging"
	"github.com/getkin/kin-openapi/openapi3"
)

// Plugin defines hooks for customizing route metadata and OpenAPI spec generation.
// Plugins can be registered globally to inject custom metadata, transform schemas,
// or modify the final OpenAPI document.
type Plugin interface {
	// Name returns the unique identifier for this plugin.
	Name() string

	// OnRouteRegister is called when a route is registered, before it's added to the registry.
	// Plugins can modify the RouteRef to add custom metadata, tags, or parameters.
	// Return an error to prevent route registration.
	OnRouteRegister(ref *RouteRef) error

	// OnSchemaGenerate is called when a schema is generated for a type.
	// Plugins can modify the schema to add custom properties, validation, or extensions.
	// The typeName parameter is the fully qualified type name (e.g., "mypackage.MyStruct").
	OnSchemaGenerate(typeName string, schema *openapi3.Schema) error

	// OnSpecBuild is called after the OpenAPI document is built, before it's returned.
	// Plugins can modify the document to add custom extensions, security schemes, or servers.
	OnSpecBuild(doc *openapi3.T) error
}

// pluginRegistry holds all registered plugins.
var pluginRegistry = &PluginRegistry{
	plugins: make(map[string]Plugin),
}

// PluginRegistry manages registered plugins.
type PluginRegistry struct {
	plugins map[string]Plugin
}

// RegisterPlugin adds a plugin to the global registry.
// If a plugin with the same name already exists, it will be replaced.
func RegisterPlugin(p Plugin) {
	if p == nil {
		return
	}
	pluginRegistry.plugins[p.Name()] = p
	logging.GetLogger().PluginRegistered(p.Name())
}

// UnregisterPlugin removes a plugin from the global registry.
func UnregisterPlugin(name string) {
	delete(pluginRegistry.plugins, name)
}

// GetPlugin retrieves a plugin by name.
func GetPlugin(name string) (Plugin, bool) {
	p, ok := pluginRegistry.plugins[name]
	return p, ok
}

// ListPlugins returns all registered plugin names.
func ListPlugins() []string {
	names := make([]string, 0, len(pluginRegistry.plugins))
	for name := range pluginRegistry.plugins {
		names = append(names, name)
	}
	return names
}

// ResetPlugins clears all registered plugins.
// This is primarily useful for testing.
func ResetPlugins() {
	pluginRegistry.plugins = make(map[string]Plugin)
}

// executeOnRouteRegister calls OnRouteRegister for all registered plugins.
func executeOnRouteRegister(ref *RouteRef) error {
	for _, plugin := range pluginRegistry.plugins {
		logging.GetLogger().PluginExecuted(plugin.Name(), "OnRouteRegister")
		if err := plugin.OnRouteRegister(ref); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteOnSchemaGenerate calls OnSchemaGenerate for all registered plugins.
// This is exported for use by the OpenAPI builder.
func ExecuteOnSchemaGenerate(typeName string, schema *openapi3.Schema) error {
	for _, plugin := range pluginRegistry.plugins {
		logging.GetLogger().PluginExecuted(plugin.Name(), "OnSchemaGenerate", "type", typeName)
		if err := plugin.OnSchemaGenerate(typeName, schema); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteOnSpecBuild calls OnSpecBuild for all registered plugins.
// This is exported for use by the OpenAPI builder.
func ExecuteOnSpecBuild(doc *openapi3.T) error {
	for _, plugin := range pluginRegistry.plugins {
		logging.GetLogger().PluginExecuted(plugin.Name(), "OnSpecBuild")
		if err := plugin.OnSpecBuild(doc); err != nil {
			return err
		}
	}
	return nil
}

// BasePlugin provides a default implementation of the Plugin interface.
// Embed this in your custom plugins to only implement the hooks you need.
type BasePlugin struct {
	PluginName string
}

// Name returns the plugin name.
func (p *BasePlugin) Name() string {
	return p.PluginName
}

// OnRouteRegister is a no-op by default.
func (p *BasePlugin) OnRouteRegister(ref *RouteRef) error {
	return nil
}

// OnSchemaGenerate is a no-op by default.
func (p *BasePlugin) OnSchemaGenerate(typeName string, schema *openapi3.Schema) error {
	return nil
}

// OnSpecBuild is a no-op by default.
func (p *BasePlugin) OnSpecBuild(doc *openapi3.T) error {
	return nil
}
