package plugins

import (
	"context"
	"net/http"

	apix "github.com/Infra-Forge/infra-apix"
	"github.com/Infra-Forge/infra-apix/openapi"
	"github.com/getkin/kin-openapi/openapi3"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id" description:"User ID"`
	Name  string `json:"name" description:"User name"`
	Email string `json:"email" description:"User email"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Name  string `json:"name" description:"User name" example:"John Doe"`
	Email string `json:"email" description:"User email" example:"john@example.com"`
}

// ExampleUsage demonstrates how to use all the plugins together
func ExampleUsage() {
	// Reset registry and plugins for clean state
	apix.ResetRegistry()
	apix.ResetPlugins()

	// 1. Register AutoTagPlugin
	autoTagPlugin := NewAutoTagPlugin(map[string]string{
		"/api/users": "users",
		"/api/posts": "posts",
		"/admin":     "admin",
	})
	apix.RegisterPlugin(autoTagPlugin)

	// 2. Register ValidationMetadataPlugin
	validationPlugin := NewValidationMetadataPlugin(true, "1.0")
	apix.RegisterPlugin(validationPlugin)

	// 3. Register CustomServersPlugin
	serversPlugin := NewCustomServersPlugin([]*openapi3.Server{
		{URL: "https://api.example.com", Description: "Production"},
		{URL: "https://staging.example.com", Description: "Staging"},
		{URL: "http://localhost:8080", Description: "Development"},
	})
	apix.RegisterPlugin(serversPlugin)

	// 4. Register SecuritySchemesPlugin
	securityPlugin := NewSecuritySchemesPlugin(
		map[string]*openapi3.SecurityScheme{
			"bearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
		},
		openapi3.SecurityRequirements{
			{"bearerAuth": {}},
		},
	)
	apix.RegisterPlugin(securityPlugin)

	// Register some routes
	apix.RegisterRoute(&apix.RouteRef{
		Method:  apix.MethodGet,
		Path:    "/api/users",
		Summary: "List all users",
		Responses: map[int]*apix.ResponseRef{
			http.StatusOK: {
				ModelType:   nil, // []User would be the actual type
				Description: "List of users",
			},
		},
	})

	apix.RegisterRoute(&apix.RouteRef{
		Method:      apix.MethodPost,
		Path:        "/api/users",
		Summary:     "Create a new user",
		RequestType: apix.TypeOf[CreateUserRequest](),
		Responses: map[int]*apix.ResponseRef{
			http.StatusCreated: {
				ModelType:   apix.TypeOf[User](),
				Description: "Created user",
			},
		},
	})

	// Build the OpenAPI spec
	builder := openapi.NewBuilder()
	builder.Title = "Example API with Plugins"
	builder.Version = "1.0.0"
	builder.Description = "This API demonstrates the use of apix plugins"

	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		panic(err)
	}

	// At this point, the document will have:
	// - Tags automatically added to routes based on path prefixes
	// - Validation metadata (x-validated, x-schema-version) on all schemas
	// - Custom servers (production, staging, development)
	// - Security schemes (bearerAuth) with global security requirements

	_ = doc // Use the document as needed
}

// Handler example
func listUsersHandler(ctx context.Context, req *apix.NoBody) ([]User, error) {
	return []User{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
	}, nil
}

func createUserHandler(ctx context.Context, req *CreateUserRequest) (User, error) {
	return User{
		ID:    1,
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

