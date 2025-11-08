package openapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apix "github.com/Infra-Forge/infra-apix"
	chiadapter "github.com/Infra-Forge/infra-apix/chi"
	echoadapter "github.com/Infra-Forge/infra-apix/echo"
	fiberadapter "github.com/Infra-Forge/infra-apix/fiber"
	ginadapter "github.com/Infra-Forge/infra-apix/gin"
	muxadapter "github.com/Infra-Forge/infra-apix/mux"
	"github.com/Infra-Forge/infra-apix/openapi"
	"github.com/Infra-Forge/infra-apix/runtime"
	"github.com/go-chi/chi/v5"
	"github.com/gofiber/fiber/v3"
	"github.com/gorilla/mux"
	"github.com/labstack/echo/v4"

	"github.com/gin-gonic/gin"
)

type Item struct {
	ID   string `json:"id"`
	Name string `json:"name" validate:"required"`
}

type CreateItemRequest struct {
	Name string `json:"name" validate:"required"`
}

type CreateItemResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func createItemHandler(ctx context.Context, req *CreateItemRequest) (CreateItemResponse, error) {
	return CreateItemResponse{ID: "item-123", Name: req.Name}, nil
}

func getItemHandler(ctx context.Context, _ *apix.NoBody) (Item, error) {
	return Item{ID: "item-123", Name: "Test Item"}, nil
}

func TestEchoIntegration(t *testing.T) {
	apix.ResetRegistry()

	e := echo.New()
	adapter := echoadapter.New(e)

	echoadapter.Post(adapter, "/api/items", createItemHandler,
		apix.WithSummary("Create item"),
		apix.WithTags("items"),
		apix.WithStandardErrors(),
	)
	echoadapter.Get(adapter, "/api/items/:id", getItemHandler,
		apix.WithSummary("Get item"),
		apix.WithTags("items"),
	)

	// Test handler execution
	req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(`{"name":"widget"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}

	// Test OpenAPI generation
	builder := openapi.NewBuilder()
	builder.Info.Title = "Echo API"
	builder.Info.Version = "1.0.0"

	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("failed to build spec: %v", err)
	}

	if doc.Paths.Len() != 2 {
		t.Fatalf("expected 2 paths, got %d", doc.Paths.Len())
	}
}

func TestChiIntegration(t *testing.T) {
	apix.ResetRegistry()

	r := chi.NewRouter()
	adapter := chiadapter.New(r)

	chiadapter.Post(adapter, "/api/items", createItemHandler,
		apix.WithSummary("Create item"),
		apix.WithTags("items"),
		apix.WithStandardErrors(),
	)
	chiadapter.Get(adapter, "/api/items/{id}", getItemHandler,
		apix.WithSummary("Get item"),
		apix.WithTags("items"),
	)

	// Test handler execution
	req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(`{"name":"widget"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}

	// Test OpenAPI generation
	builder := openapi.NewBuilder()
	builder.Info.Title = "Chi API"
	builder.Info.Version = "1.0.0"

	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("failed to build spec: %v", err)
	}

	if doc.Paths.Len() != 2 {
		t.Fatalf("expected 2 paths, got %d", doc.Paths.Len())
	}
}

func TestMuxIntegration(t *testing.T) {
	apix.ResetRegistry()

	r := mux.NewRouter()
	adapter := muxadapter.New(r)

	muxadapter.Post(adapter, "/api/items", createItemHandler,
		apix.WithSummary("Create item"),
		apix.WithTags("items"),
		apix.WithStandardErrors(),
	)
	muxadapter.Get(adapter, "/api/items/{id}", getItemHandler,
		apix.WithSummary("Get item"),
		apix.WithTags("items"),
	)

	// Test handler execution
	req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(`{"name":"widget"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}

	// Test OpenAPI generation
	builder := openapi.NewBuilder()
	builder.Info.Title = "Mux API"
	builder.Info.Version = "1.0.0"

	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("failed to build spec: %v", err)
	}

	if doc.Paths.Len() != 2 {
		t.Fatalf("expected 2 paths, got %d", doc.Paths.Len())
	}
}

func TestGinIntegration(t *testing.T) {
	apix.ResetRegistry()
	gin.SetMode(gin.TestMode)

	e := gin.New()
	adapter := ginadapter.New(e)

	ginadapter.Post(adapter, "/api/items", createItemHandler,
		apix.WithSummary("Create item"),
		apix.WithTags("items"),
		apix.WithStandardErrors(),
	)
	ginadapter.Get(adapter, "/api/items/:id", getItemHandler,
		apix.WithSummary("Get item"),
		apix.WithTags("items"),
	)

	// Test handler execution
	req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(`{"name":"widget"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}

	// Test OpenAPI generation
	builder := openapi.NewBuilder()
	builder.Info.Title = "Gin API"
	builder.Info.Version = "1.0.0"

	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("failed to build spec: %v", err)
	}

	if doc.Paths.Len() != 2 {
		t.Fatalf("expected 2 paths, got %d", doc.Paths.Len())
	}
}

func TestFiberIntegration(t *testing.T) {
	apix.ResetRegistry()

	app := fiber.New()
	adapter := fiberadapter.New(app)

	fiberadapter.Post(adapter, "/api/items", createItemHandler,
		apix.WithSummary("Create item"),
		apix.WithTags("items"),
		apix.WithStandardErrors(),
	)
	fiberadapter.Get(adapter, "/api/items/:id", getItemHandler,
		apix.WithSummary("Get item"),
		apix.WithTags("items"),
	)

	// Test handler execution
	req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(`{"name":"widget"}`))
	req.Header.Set("Content-Type", "application/json")
	fiberResp, err := app.Test(req)
	if err != nil {
		t.Fatalf("fiber test failed: %v", err)
	}

	if fiberResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", fiberResp.StatusCode)
	}

	// Test OpenAPI generation
	builder := openapi.NewBuilder()
	builder.Info.Title = "Fiber API"
	builder.Info.Version = "1.0.0"

	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("failed to build spec: %v", err)
	}

	if doc.Paths.Len() != 2 {
		t.Fatalf("expected 2 paths, got %d", doc.Paths.Len())
	}
}

func TestRuntimeHandlerIntegration(t *testing.T) {
	apix.ResetRegistry()

	e := echo.New()
	adapter := echoadapter.New(e)

	echoadapter.Get(adapter, "/api/health", func(ctx context.Context, _ *apix.NoBody) (map[string]string, error) {
		return map[string]string{"status": "ok"}, nil
	})

	// Create runtime handler
	handler, err := runtime.NewHandler(runtime.Config{
		Title:   "Test API",
		Version: "1.0.0",
		Format:  "json",
	})
	if err != nil {
		t.Fatalf("failed to create runtime handler: %v", err)
	}

	handler.RegisterEcho(e)

	// Test OpenAPI endpoint
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var spec map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &spec); err != nil {
		t.Fatalf("failed to parse spec: %v", err)
	}

	if spec["openapi"] != "3.1.0" {
		t.Fatalf("expected OpenAPI 3.1.0")
	}
}
