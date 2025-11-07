package chi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Infra-Forge/apix"
	chiadapter "github.com/Infra-Forge/apix/chi"
	"github.com/go-chi/chi/v5"
)

type createItemRequest struct {
	Name string `json:"name"`
}

type createItemResponse struct {
	ID string `json:"id"`
}

func TestChiAdapterRegistersAndHandles(t *testing.T) {
	apix.ResetRegistry()

	r := chi.NewRouter()
	adapter := chiadapter.New(r)

	var capturedReq *createItemRequest
	chiadapter.Post(adapter, "/api/items", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
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

func TestChiAdapterDefaultDecoderValidation(t *testing.T) {
	apix.ResetRegistry()
	r := chi.NewRouter()
	adapter := chiadapter.New(r, chiadapter.Options{
		Validator: &mockValidator{},
	})

	chiadapter.Post(adapter, "/validate", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
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

func TestChiAdapterMethodHelpers(t *testing.T) {
	apix.ResetRegistry()
	r := chi.NewRouter()
	adapter := chiadapter.New(r)

	chiadapter.Get(adapter, "/method/get", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{ID: "g"}, nil
	})
	chiadapter.Put(adapter, "/method/put", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})
	chiadapter.Patch(adapter, "/method/patch", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})
	chiadapter.Delete(adapter, "/method/delete", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
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

func TestChiAdapterCustomErrorHandler(t *testing.T) {
	apix.ResetRegistry()
	r := chi.NewRouter()
	adapter := chiadapter.New(r, chiadapter.Options{
		ErrorHandler: func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusTeapot)
		},
	})

	chiadapter.Get(adapter, "/fail", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{}, fmt.Errorf("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusTeapot {
		t.Fatalf("expected custom error status, got %d", resp.Code)
	}
}

func TestChiAdapterCustomResponseEncoder(t *testing.T) {
	apix.ResetRegistry()
	r := chi.NewRouter()
	adapter := chiadapter.New(r, chiadapter.Options{
		ResponseEncoder: func(ctx context.Context, w http.ResponseWriter, r *http.Request, status int, payload any, ref *apix.RouteRef) error {
			w.WriteHeader(status)
			_, err := w.Write([]byte("custom"))
			return err
		},
	})

	chiadapter.Post(adapter, "/custom", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
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

func TestChiAdapterCustomDecoder(t *testing.T) {
	apix.ResetRegistry()
	r := chi.NewRouter()
	adapter := chiadapter.New(r, chiadapter.Options{
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
	chiadapter.Post(adapter, "/custom-decode", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
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

func TestChiAdapterDecoderErrors(t *testing.T) {
	apix.ResetRegistry()
	r := chi.NewRouter()
	adapter := chiadapter.New(r)

	chiadapter.Post(adapter, "/decode-test", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
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

func TestChiAdapterRouteOptions(t *testing.T) {
	apix.ResetRegistry()
	r := chi.NewRouter()
	adapter := chiadapter.New(r)

	chiadapter.Post(adapter, "/options-test", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
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

type mockValidator struct{}

func (m *mockValidator) Validate(i any) error {
	return fmt.Errorf("invalid")
}
