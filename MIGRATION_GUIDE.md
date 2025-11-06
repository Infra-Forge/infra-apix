# Migration Guide: From swaggo/swag to apix

This guide shows how to migrate your `infranotes-module` project from `swaggo/swag` (annotation-based) to `apix` (code-first).

## Why Migrate?

### Current Pain Points (swaggo/swag)
- ❌ Manual annotations in comments (`// @Summary`, `// @Param`, etc.)
- ❌ Requires running `swag init` to generate docs
- ❌ Annotations get out of sync with actual code
- ❌ No compile-time type safety
- ❌ Difficult to maintain as API grows
- ❌ Swagger UI served from static files

### Benefits of apix
- ✅ **Type-safe**: Uses actual Go types, no manual annotations
- ✅ **Auto-generated**: OpenAPI spec generated at runtime from your types
- ✅ **Always in sync**: Spec reflects actual code
- ✅ **No build step**: No need to run `swag init`
- ✅ **Runtime flexibility**: Spec updates automatically when code changes
- ✅ **Built-in Swagger UI**: No need to manage static files

## Step-by-Step Migration

### Step 1: Install apix

```bash
cd ../infranotes-module
go get github.com/Infra-Forge/apix
```

Update `go.mod`:
```go
require (
    github.com/Infra-Forge/apix v1.0.0
    // Keep existing dependencies
    github.com/labstack/echo/v4 v4.13.3
    github.com/google/uuid v1.6.0
    github.com/shopspring/decimal v1.4.0
)
```

### Step 2: Define Your Models (You Already Have These!)

Your existing models work as-is:

```go
// internal/analytics/document/models/document.go
type DocumentModel struct {
    ID            uuid.UUID  `json:"id"`
    FileName      string     `json:"file_name"`
    ContentType   string     `json:"content_type"`
    FileSize      int64      `json:"file_size"`
    ProcessStatus string     `json:"process_status"`
    ProcessedAt   *time.Time `json:"processed_at,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
}

type DocumentListResponse struct {
    Data       []DocumentModel `json:"data"`
    Pagination PaginationMeta  `json:"pagination"`
}
```

**No changes needed!** apix uses your existing structs.

### Step 3: Convert Handlers

#### Before (swaggo/swag):
```go
// @Summary Upload financial document
// @Description Upload and process a financial document (PDF, CSV, XLSX)
// @Tags Documents
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Document file"
// @Success 201 {object} DocumentModel
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/documents [post]
func (h *DocumentHandler) UploadDocument(c echo.Context) error {
    // ... handler logic ...
    return c.JSON(http.StatusCreated, document)
}
```

#### After (apix):
```go
import (
    "github.com/Infra-Forge/apix"
    echoadapter "github.com/Infra-Forge/apix/echo"
)

// Handler function with type-safe signature
func uploadDocument(ctx context.Context, req apix.NoBody) (DocumentModel, error) {
    // ... handler logic ...
    return document, nil
}

// Registration in main.go
echoadapter.Post(adapter, "/api/documents", uploadDocument,
    apix.WithSummary("Upload financial document"),
    apix.WithDescription("Upload and process a financial document (PDF, CSV, XLSX)"),
    apix.WithTags("Documents"),
    apix.WithSecurity("BearerAuth", "documents:write"),
    apix.WithCreatedStatus(), // Returns 201
)
```

### Step 4: Update main.go

#### Remove swaggo imports:
```go
// DELETE THESE:
// _ "github.com/StackCatalyst/infranotes-module/docs/swagger"
// "github.com/swaggo/echo-swagger"
// "github.com/swaggo/files"
```

#### Add apix imports:
```go
import (
    "github.com/Infra-Forge/apix"
    echoadapter "github.com/Infra-Forge/apix/echo"
    "github.com/Infra-Forge/apix/runtime"
)
```

#### Replace Swagger setup:
```go
// DELETE THIS:
e.Static("/swagger", "docs/swagger")
e.GET("/swagger/", func(c echo.Context) error {
    return c.File("docs/swagger/index.html")
})

// REPLACE WITH:
// Create apix adapter
adapter := echoadapter.New(e)

// Register all your routes (see Step 5)

// Setup OpenAPI runtime handler
handler, err := runtime.NewHandler(runtime.Config{
    Title:           "InfraNotes Financial Analytics API",
    Version:         "1.0.0",
    Format:          "json",
    Servers:         []string{"https://api.infranotes.io", "http://localhost:8080"},
    EnableSwaggerUI: true,
    CacheTTL:        5 * time.Minute, // Cache spec for 5 minutes
    CustomizeBuilder: func(b *openapi.Builder) {
        b.Info.Description = "Comprehensive financial analytics and document processing API"
        b.SecuritySchemes = openapi3.SecuritySchemes{
            "BearerAuth": &openapi3.SecuritySchemeRef{
                Value: openapi3.NewJWTSecurityScheme(),
            },
        }
        b.Info.Contact = &openapi3.Contact{
            Name:  "InfraNotes API Support",
            Email: "support@infranotes.io",
        }
    },
})
if err != nil {
    log.Fatalf("Failed to create OpenAPI handler: %v", err)
}

// Register OpenAPI endpoints
handler.RegisterEcho(e)
// Now available at:
// - GET /openapi.json (spec)
// - GET /swagger (Swagger UI)
```

### Step 5: Register Your Routes

Create a registration function:

```go
func registerRoutes(adapter *echoadapter.EchoAdapter, deps *Dependencies) {
    // Document routes
    echoadapter.Post(adapter, "/api/documents", deps.uploadDocument,
        apix.WithSummary("Upload financial document"),
        apix.WithTags("Documents"),
        apix.WithSecurity("BearerAuth"),
    )
    
    echoadapter.Get(adapter, "/api/documents/:id", deps.getDocument,
        apix.WithSummary("Get document by ID"),
        apix.WithTags("Documents"),
        apix.WithParameter(apix.Parameter{
            Name:        "id",
            In:          "path",
            Required:    true,
            SchemaType:  "string",
            Description: "Document UUID",
        }),
    )
    
    echoadapter.Get(adapter, "/api/documents", deps.listDocuments,
        apix.WithSummary("List all documents"),
        apix.WithTags("Documents"),
        apix.WithParameter(apix.Parameter{
            Name: "page", In: "query", SchemaType: "integer",
        }),
        apix.WithParameter(apix.Parameter{
            Name: "per_page", In: "query", SchemaType: "integer",
        }),
    )
    
    // Transaction routes
    echoadapter.Post(adapter, "/api/transactions", deps.createTransaction,
        apix.WithSummary("Create transaction"),
        apix.WithTags("Transactions"),
        apix.WithSecurity("BearerAuth"),
    )
    
    // ... more routes ...
}
```

### Step 6: Remove Build Step

Delete from your Makefile or build scripts:
```bash
# DELETE THIS:
swag init -g cmd/financial-analytics/main.go -o docs/swagger
```

You no longer need to generate docs!

### Step 7: Test

```bash
# Run your application
go run cmd/financial-analytics/main.go

# Access Swagger UI
open http://localhost:8080/swagger

# Get OpenAPI spec
curl http://localhost:8080/openapi.json
```

## Complete Example

See `examples/infranotes/` in the apix repository for a complete working example that mirrors your project structure.

## Migration Checklist

- [ ] Install apix: `go get github.com/Infra-Forge/apix`
- [ ] Remove swaggo dependencies from go.mod
- [ ] Remove swaggo imports from main.go
- [ ] Create apix adapter: `adapter := echoadapter.New(e)`
- [ ] Convert handlers to type-safe functions
- [ ] Register routes with apix options
- [ ] Setup runtime OpenAPI handler
- [ ] Remove `swag init` from build process
- [ ] Delete `docs/swagger` directory
- [ ] Test Swagger UI at `/swagger`
- [ ] Verify OpenAPI spec at `/openapi.json`

## Gradual Migration

You can migrate gradually:

1. **Keep swaggo running** for existing routes
2. **Add new routes** using apix
3. **Migrate old routes** one at a time
4. **Remove swaggo** when all routes are migrated

Both can coexist temporarily:
```go
// Old swaggo route
e.GET("/old-endpoint", oldHandler)

// New apix route
echoadapter.Get(adapter, "/new-endpoint", newHandler,
    apix.WithSummary("New endpoint"),
)
```

## What You Get

After migration:
- ✅ No more manual annotations
- ✅ No more `swag init` build step
- ✅ Type-safe handlers
- ✅ Auto-generated OpenAPI spec
- ✅ Built-in Swagger UI
- ✅ Spec always in sync with code
- ✅ Better developer experience

## Need Help?

See the complete working example:
```bash
cd ../infra-apix/examples/infranotes
go run main.go
```

This example has the exact same structure as your project with:
- Documents, Transactions, Categories
- Nested response types
- Pagination
- UUID, decimal.Decimal, time.Time types
- Security schemes

