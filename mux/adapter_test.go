package mux_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apix "github.com/Infra-Forge/infra-apix"
	muxadapter "github.com/Infra-Forge/infra-apix/mux"
	"github.com/gorilla/mux"
)

type createItemRequest struct {
	Name string `json:"name"`
}

type createItemResponse struct {
	ID string `json:"id"`
}

func TestMuxAdapterRegistersAndHandles(t *testing.T) {
	apix.ResetRegistry()

	r := mux.NewRouter()
	adapter := muxadapter.New(r)

	var capturedReq *createItemRequest
	muxadapter.Post(adapter, "/api/items", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		capturedReq = req
		return createItemResponse{ID: "123"}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(`{"name":"widget"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.Code)
	}
	if capturedReq == nil || capturedReq.Name != "widget" {
		t.Fatalf("handler did not receive decoded request")
	}
	var decoded createItemResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
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

func TestMuxAdapterDefaultDecoderValidation(t *testing.T) {
	apix.ResetRegistry()
	r := mux.NewRouter()
	adapter := muxadapter.New(r, muxadapter.Options{
		Validator: &mockValidator{},
	})

	muxadapter.Post(adapter, "/validate", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: "ok"}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation failure, got %d", resp.Code)
	}
}

func TestMuxAdapterMethodHelpers(t *testing.T) {
	apix.ResetRegistry()
	r := mux.NewRouter()
	adapter := muxadapter.New(r)

	muxadapter.Get(adapter, "/method/get", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{ID: "g"}, nil
	})
	muxadapter.Put(adapter, "/method/put", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})
	muxadapter.Patch(adapter, "/method/patch", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})
	muxadapter.Delete(adapter, "/method/delete", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
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
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)
		if resp.Code != tc.expect {
			t.Fatalf("%s %s expected %d, got %d", tc.method, tc.path, tc.expect, resp.Code)
		}
	}

	if len(apix.Snapshot()) != 4 {
		t.Fatalf("expected 4 routes registered")
	}
}

func TestMuxAdapterCustomErrorHandler(t *testing.T) {
	apix.ResetRegistry()
	r := mux.NewRouter()
	adapter := muxadapter.New(r, muxadapter.Options{
		ErrorHandler: func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusTeapot)
		},
	})

	muxadapter.Get(adapter, "/fail", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{}, fmt.Errorf("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusTeapot {
		t.Fatalf("expected custom error status, got %d", resp.Code)
	}
}

func TestMuxAdapterCustomResponseEncoder(t *testing.T) {
	apix.ResetRegistry()
	r := mux.NewRouter()
	adapter := muxadapter.New(r, muxadapter.Options{
		ResponseEncoder: func(ctx context.Context, w http.ResponseWriter, r *http.Request, status int, payload any, ref *apix.RouteRef) error {
			w.WriteHeader(status)
			_, err := w.Write([]byte("custom"))
			return err
		},
	})

	muxadapter.Post(adapter, "/custom", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/custom", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	if strings.TrimSpace(resp.Body.String()) != "custom" {
		t.Fatalf("expected custom encoder output")
	}
}

func TestMuxAdapterCustomDecoder(t *testing.T) {
	apix.ResetRegistry()
	r := mux.NewRouter()
	adapter := muxadapter.New(r, muxadapter.Options{
		Decoder: func(ctx context.Context, w http.ResponseWriter, r *http.Request, dst any) error {
			req, ok := dst.(*createItemRequest)
			if !ok {
				return fmt.Errorf("unexpected type")
			}
			req.Name = "custom-decoded"
			return nil
		},
	})

	var capturedReq *createItemRequest
	muxadapter.Post(adapter, "/custom-decode", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		capturedReq = req
		return createItemResponse{ID: req.Name}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/custom-decode", strings.NewReader(`{"name":"ignored"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	if capturedReq == nil || capturedReq.Name != "custom-decoded" {
		t.Fatalf("expected custom decoder to be used")
	}
}

func TestMuxAdapterDecoderErrors(t *testing.T) {
	apix.ResetRegistry()
	r := mux.NewRouter()
	adapter := muxadapter.New(r)

	muxadapter.Post(adapter, "/decode-test", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
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
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			if resp.Code != tc.expect {
				t.Fatalf("%s: expected %d, got %d", tc.name, tc.expect, resp.Code)
			}
		})
	}
}

func TestMuxAdapterRouteOptions(t *testing.T) {
	apix.ResetRegistry()
	r := mux.NewRouter()
	adapter := muxadapter.New(r)

	muxadapter.Post(adapter, "/options-test", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
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

func TestMuxAdapterPathParameters(t *testing.T) {
	apix.ResetRegistry()
	r := mux.NewRouter()
	adapter := muxadapter.New(r)

	muxadapter.Get(adapter, "/items/{id}", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{ID: "found"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/items/123", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var decoded createItemResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("response not json: %v", err)
	}
	if decoded.ID != "found" {
		t.Fatalf("unexpected response")
	}
}

type mockValidator struct{}

func (m *mockValidator) Validate(i any) error {
	return fmt.Errorf("invalid")
}

func TestMuxAdapterProblemDetailsEncoding(t *testing.T) {
	apix.ResetRegistry()

	r := mux.NewRouter()
	adapter := muxadapter.New(r, muxadapter.Options{
		UseProblemDetails: true,
	})

	muxadapter.Get(adapter, "/test", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{}, apix.NotFound("user not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.Code)
	}

	contentType := resp.Header().Get("Content-Type")
	if contentType != "application/problem+json" {
		t.Fatalf("expected Content-Type application/problem+json, got %s", contentType)
	}

	var problem map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to decode problem details: %v", err)
	}

	if problem["status"] != float64(404) {
		t.Errorf("expected status 404, got %v", problem["status"])
	}
	if problem["title"] != "Not Found" {
		t.Errorf("expected title 'Not Found', got %v", problem["title"])
	}
	if problem["detail"] != "user not found" {
		t.Errorf("expected detail 'user not found', got %v", problem["detail"])
	}
}
