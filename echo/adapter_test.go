package echo_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apix "github.com/Infra-Forge/infra-apix"
	echoadapter "github.com/Infra-Forge/infra-apix/echo"
	"github.com/labstack/echo/v4"
)

type createItemRequest struct {
	Name string `json:"name"`
}

type createItemResponse struct {
	ID string `json:"id"`
}

func TestEchoAdapterRegistersAndHandles(t *testing.T) {
	apix.ResetRegistry()

	e := echo.New()
	adapter := echoadapter.New(e)

	var capturedReq *createItemRequest
	echoadapter.Post(adapter, "/api/items", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		capturedReq = req
		return createItemResponse{ID: "123"}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/items", strings.NewReader(`{"name":"widget"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()

	e.ServeHTTP(resp, req)

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

func TestEchoAdapterDefaultDecoderValidation(t *testing.T) {
	apix.ResetRegistry()
	e := echo.New()
	e.Validator = &mockValidator{}
	adapter := echoadapter.New(e)

	echoadapter.Post(adapter, "/validate", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: "ok"}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader("{}"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected validation failure, got %d", resp.Code)
	}
}

func TestEchoAdapterMethodHelpers(t *testing.T) {
	apix.ResetRegistry()
	e := echo.New()
	adapter := echoadapter.New(e)

	echoadapter.Get(adapter, "/method/get", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{ID: "g"}, nil
	})
	echoadapter.Put(adapter, "/method/put", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})
	echoadapter.Patch(adapter, "/method/patch", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})
	echoadapter.Delete(adapter, "/method/delete", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
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
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		}
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		if resp.Code != tc.expect {
			t.Fatalf("%s %s expected %d, got %d", tc.method, tc.path, tc.expect, resp.Code)
		}
	}

	if len(apix.Snapshot()) != 4 {
		t.Fatalf("expected 4 routes registered")
	}
}

func TestEchoAdapterErrorTransformer(t *testing.T) {
	apix.ResetRegistry()
	e := echo.New()
	adapter := echoadapter.New(e, echoadapter.Options{
		ErrorHandler: func(err error) error {
			return echo.NewHTTPError(http.StatusTeapot, err.Error())
		},
	})

	echoadapter.Get(adapter, "/fail", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{}, fmt.Errorf("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	if resp.Code != http.StatusTeapot {
		t.Fatalf("expected transformed error status, got %d", resp.Code)
	}
}

func TestEchoAdapterCustomResponseEncoder(t *testing.T) {
	apix.ResetRegistry()
	e := echo.New()
	adapter := echoadapter.New(e, echoadapter.Options{
		ResponseEncoder: func(ctx context.Context, c echo.Context, status int, payload any, ref *apix.RouteRef) error {
			return c.String(status, "custom")
		},
	})

	echoadapter.Post(adapter, "/custom", func(ctx context.Context, req *createItemRequest) (createItemResponse, error) {
		return createItemResponse{ID: req.Name}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/custom", strings.NewReader(`{"name":"x"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
	if strings.TrimSpace(resp.Body.String()) != "custom" {
		t.Fatalf("expected custom encoder output")
	}
}

type mockValidator struct{}

func (m *mockValidator) Validate(i any) error {
	return fmt.Errorf("invalid")
}

func TestEchoAdapterProblemDetailsEncoding(t *testing.T) {
	apix.ResetRegistry()

	e := echo.New()
	adapter := echoadapter.New(e, echoadapter.Options{
		UseProblemDetails: true,
	})

	// Install the ProblemDetails error handler
	e.HTTPErrorHandler = echoadapter.ProblemDetailsErrorHandler(e.HTTPErrorHandler)

	echoadapter.Get(adapter, "/test", func(ctx context.Context, _ *apix.NoBody) (createItemResponse, error) {
		return createItemResponse{}, apix.NotFound("user not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	e.ServeHTTP(resp, req)

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
