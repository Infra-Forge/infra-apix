package apix_test

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	apix "github.com/Infra-Forge/infra-apix"
	"github.com/Infra-Forge/infra-apix/openapi"
	"github.com/getkin/kin-openapi/openapi3"
)

// TestPluginRegistration tests plugin registration and retrieval
func TestPluginRegistration(t *testing.T) {
	apix.ResetPlugins()
	defer apix.ResetPlugins()

	plugin := &apix.BasePlugin{PluginName: "test-plugin"}
	apix.RegisterPlugin(plugin)

	retrieved, ok := apix.GetPlugin("test-plugin")
	if !ok {
		t.Fatal("expected plugin to be registered")
	}

	if retrieved.Name() != "test-plugin" {
		t.Errorf("expected plugin name 'test-plugin', got %s", retrieved.Name())
	}

	// Test ListPlugins
	plugins := apix.ListPlugins()
	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(plugins))
	}

	// Test UnregisterPlugin
	apix.UnregisterPlugin("test-plugin")
	_, ok = apix.GetPlugin("test-plugin")
	if ok {
		t.Fatal("expected plugin to be unregistered")
	}
}

// AutoTagPlugin automatically adds tags to routes based on path prefix
type AutoTagPlugin struct {
	apix.BasePlugin
}

func (p *AutoTagPlugin) OnRouteRegister(ref *apix.RouteRef) error {
	// Auto-tag routes based on path prefix
	if len(ref.Path) > 5 && ref.Path[:5] == "/api/" {
		ref.Tags = append(ref.Tags, "api")
	}
	if len(ref.Path) > 7 && ref.Path[:7] == "/admin/" {
		ref.Tags = append(ref.Tags, "admin")
	}
	return nil
}

func TestPluginOnRouteRegister(t *testing.T) {
	apix.ResetRegistry()
	apix.ResetPlugins()
	defer apix.ResetPlugins()

	plugin := &AutoTagPlugin{
		BasePlugin: apix.BasePlugin{PluginName: "auto-tag"},
	}
	apix.RegisterPlugin(plugin)

	ref := &apix.RouteRef{
		Method:      apix.MethodGet,
		Path:        "/api/users",
		Summary:     "List users",
		RequestType: nil,
		Responses: map[int]*apix.ResponseRef{
			http.StatusOK: {ModelType: nil},
		},
	}

	apix.RegisterRoute(ref)

	// Verify tags were added
	if len(ref.Tags) != 1 || ref.Tags[0] != "api" {
		t.Errorf("expected tags ['api'], got %v", ref.Tags)
	}
}

// ValidationPlugin adds validation metadata to schemas
type ValidationPlugin struct {
	apix.BasePlugin
}

func (p *ValidationPlugin) OnSchemaGenerate(typeName string, schema *openapi3.Schema) error {
	// Add custom extension for validation
	if schema.Extensions == nil {
		schema.Extensions = make(map[string]interface{})
	}
	schema.Extensions["x-validated"] = true
	return nil
}

func TestPluginOnSchemaGenerate(t *testing.T) {
	apix.ResetRegistry()
	apix.ResetPlugins()
	defer apix.ResetPlugins()

	plugin := &ValidationPlugin{
		BasePlugin: apix.BasePlugin{PluginName: "validation"},
	}
	apix.RegisterPlugin(plugin)

	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	ref := &apix.RouteRef{
		Method:      apix.MethodPost,
		Path:        "/api/users",
		Summary:     "Create user",
		RequestType: reflect.TypeOf(User{}),
		Responses: map[int]*apix.ResponseRef{
			http.StatusCreated: {ModelType: reflect.TypeOf(User{})},
		},
	}

	apix.RegisterRoute(ref)

	builder := openapi.NewBuilder()
	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify extension was added to schema
	// The component name should be the package path + type name
	var userSchema *openapi3.SchemaRef
	for name, schema := range doc.Components.Schemas {
		if schema != nil && schema.Value != nil {
			// Find the User schema (should contain name and email properties)
			if _, hasName := schema.Value.Properties["name"]; hasName {
				if _, hasEmail := schema.Value.Properties["email"]; hasEmail {
					userSchema = schema
					t.Logf("Found User schema with name: %s", name)
					break
				}
			}
		}
	}

	if userSchema == nil || userSchema.Value == nil {
		t.Fatal("User schema not found")
	}

	validated, ok := userSchema.Value.Extensions["x-validated"]
	if !ok || validated != true {
		t.Errorf("expected x-validated extension to be true, got %v", validated)
	}
}

// CustomServerPlugin adds custom servers to the spec
type CustomServerPlugin struct {
	apix.BasePlugin
}

func (p *CustomServerPlugin) OnSpecBuild(doc *openapi3.T) error {
	doc.Servers = append(doc.Servers, &openapi3.Server{
		URL:         "https://api.example.com",
		Description: "Production server",
	})
	return nil
}

func TestPluginOnSpecBuild(t *testing.T) {
	apix.ResetRegistry()
	apix.ResetPlugins()
	defer apix.ResetPlugins()

	plugin := &CustomServerPlugin{
		BasePlugin: apix.BasePlugin{PluginName: "custom-server"},
	}
	apix.RegisterPlugin(plugin)

	ref := &apix.RouteRef{
		Method:  apix.MethodGet,
		Path:    "/health",
		Summary: "Health check",
		Responses: map[int]*apix.ResponseRef{
			http.StatusOK: {ModelType: nil},
		},
	}

	apix.RegisterRoute(ref)

	builder := openapi.NewBuilder()
	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify server was added
	if len(doc.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(doc.Servers))
	}

	if doc.Servers[0].URL != "https://api.example.com" {
		t.Errorf("expected server URL 'https://api.example.com', got %s", doc.Servers[0].URL)
	}
}

// ErrorPlugin returns an error during route registration
type ErrorPlugin struct {
	apix.BasePlugin
}

func (p *ErrorPlugin) OnRouteRegister(ref *apix.RouteRef) error {
	return errors.New("plugin error")
}

func TestPluginErrorHandling(t *testing.T) {
	apix.ResetRegistry()
	apix.ResetPlugins()
	defer apix.ResetPlugins()

	plugin := &ErrorPlugin{
		BasePlugin: apix.BasePlugin{PluginName: "error-plugin"},
	}
	apix.RegisterPlugin(plugin)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from plugin error")
		}
	}()

	ref := &apix.RouteRef{
		Method:  apix.MethodGet,
		Path:    "/test",
		Summary: "Test",
		Responses: map[int]*apix.ResponseRef{
			http.StatusOK: {ModelType: nil},
		},
	}

	apix.RegisterRoute(ref)
}
