package fiber_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Infra-Forge/apix"
	fiberadapter "github.com/Infra-Forge/apix/fiber"
	"github.com/gofiber/fiber/v3"
)

type createItemRequest struct {
	Name string `json:"name"`
}

type createItemResponse struct {
	ID string `json:"id"`
}

func TestFiberAdapterRegistersAndHandles(t *testing.T) {
	apix.ResetRegistry()

	app := fiber.New()
	adapter := fiberadapter.New(app)

	var capturedReq *createItemRequest
	fiberadapter.Post(adapter, "/api/items", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		capturedReq = req
		return createItemResponse{ID: "123"}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(`{"name":"widget"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}
	if capturedReq == nil || capturedReq.Name != "widget" {
		t.Fatalf("handler did not receive decoded request")
	}
	body, _ := io.ReadAll(resp.Body)
	var decoded createItemResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("response not json: %v", err)
	}
	if decoded.ID != "123" {
		t.Fatalf("unexpected response payload")
	}

	snapshot := apix.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected route registered in registry")
	}
	if snapshot[0].Path != "/api/items" {
		t.Fatalf("expected path stored")
	}
}

func TestFiberAdapterMethodHelpers(t *testing.T) {
	apix.ResetRegistry()
	app := fiber.New()
	adapter := fiberadapter.New(app)

	fiberadapter.Get(adapter, "/method/get", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{ID: "g"}, nil
	})
	fiberadapter.Put(adapter, "/method/put", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})
	fiberadapter.Patch(adapter, "/method/patch", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})
	fiberadapter.Delete(adapter, "/method/delete", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{}, nil
	})

	cases := []struct {
		method string
		path   string
		body   string
		expect int
	}{
		{http.MethodGet, "/method/get", "", http.StatusOK},
		{http.MethodPut, "/method/put", `{"name":"put"}`, http.StatusOK},
		{http.MethodPatch, "/method/patch", `{"name":"patch"}`, http.StatusOK},
		{http.MethodDelete, "/method/delete", "", http.StatusNoContent},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != tc.expect {
			t.Fatalf("%s %s expected %d, got %d", tc.method, tc.path, tc.expect, resp.StatusCode)
		}
	}

	if len(apix.Snapshot()) != 4 {
		t.Fatalf("expected 4 routes registered")
	}
}

func TestFiberAdapterCustomErrorHandler(t *testing.T) {
	apix.ResetRegistry()
	app := fiber.New()
	adapter := fiberadapter.New(app, fiberadapter.Options{
		ErrorHandler: func(ctx context.Context, c fiber.Ctx, err error) error {
			return c.Status(http.StatusTeapot).JSON(fiber.Map{"error": err.Error()})
		},
	})

	fiberadapter.Get(adapter, "/fail", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{}, fmt.Errorf("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != http.StatusTeapot {
		t.Fatalf("expected custom error status, got %d", resp.StatusCode)
	}
}

func TestFiberAdapterCustomResponseEncoder(t *testing.T) {
	apix.ResetRegistry()
	app := fiber.New()
	adapter := fiberadapter.New(app, fiberadapter.Options{
		ResponseEncoder: func(ctx context.Context, c fiber.Ctx, status int, payload any, ref *apix.RouteRef) error {
			return c.Status(status).SendString("custom")
		},
	})

	fiberadapter.Post(adapter, "/custom", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/custom", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if strings.TrimSpace(string(body)) != "custom" {
		t.Fatalf("expected custom encoder output")
	}
}

func TestFiberAdapterCustomDecoder(t *testing.T) {
	apix.ResetRegistry()
	app := fiber.New()
	adapter := fiberadapter.New(app, fiberadapter.Options{
		Decoder: func(ctx context.Context, c fiber.Ctx, dst any) error {
			req, ok := dst.(*createItemRequest)
			if !ok {
				return fmt.Errorf("unexpected type")
			}
			req.Name = "custom-decoded"
			return nil
		},
	})

	var capturedReq *createItemRequest
	fiberadapter.Post(adapter, "/custom-decode", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		capturedReq = req
		return createItemResponse{ID: req.Name}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/custom-decode", strings.NewReader(`{"name":"ignored"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if capturedReq == nil || capturedReq.Name != "custom-decoded" {
		t.Fatalf("expected custom decoder to be used")
	}
}

func TestFiberAdapterDecoderErrors(t *testing.T) {
	apix.ResetRegistry()
	app := fiber.New()
	adapter := fiberadapter.New(app)

	fiberadapter.Post(adapter, "/decode-test", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: "ok"}, nil
	})

	cases := []struct {
		name   string
		body   string
		expect int
	}{
		{"empty body", "", http.StatusBadRequest},
		{"invalid json", `{"name":`, http.StatusBadRequest},
		{"unknown fields", `{"name":"test","unknown":"field"}`, http.StatusBadRequest},
		{"extra content", `{"name":"test"}{"extra":"data"}`, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/decode-test", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("test request failed: %v", err)
			}
			if resp.StatusCode != tc.expect {
				t.Fatalf("%s: expected %d, got %d", tc.name, tc.expect, resp.StatusCode)
			}
		})
	}
}

func TestFiberAdapterRouteOptions(t *testing.T) {
	apix.ResetRegistry()
	app := fiber.New()
	adapter := fiberadapter.New(app)

	fiberadapter.Post(adapter, "/options-test", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: "123"}, nil
	},
		apix.WithSummary("Create item"),
		apix.WithDescription("Creates a new item"),
		apix.WithTags("items", "v1"),
		apix.WithSecurity("BearerAuth"),
		apix.WithParameter(apix.Parameter{Name: "X-Request-ID", In: "header", SchemaType: "string"}),
	)

	snapshot := apix.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 route")
	}

	route := snapshot[0]
	if route.Summary != "Create item" {
		t.Fatalf("expected summary set")
	}
	if route.Description != "Creates a new item" {
		t.Fatalf("expected description set")
	}
	if len(route.Tags) != 2 {
		t.Fatalf("expected 2 tags")
	}
	if len(route.Security) != 1 {
		t.Fatalf("expected security requirement")
	}
	if len(route.Parameters) != 1 {
		t.Fatalf("expected parameter")
	}
}

func TestFiberAdapterPathParameters(t *testing.T) {
	apix.ResetRegistry()
	app := fiber.New()
	adapter := fiberadapter.New(app)

	fiberadapter.Get(adapter, "/items/:id", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{ID: "found"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var decoded createItemResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("response not json: %v", err)
	}
	if decoded.ID != "found" {
		t.Fatalf("unexpected response")
	}
}

