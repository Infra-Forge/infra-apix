package apix_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/Infra-Forge/apix"
)

type sampleReq struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age,omitempty"`
}

type sampleResp struct {
	ID string `json:"id"`
}

func TestRouteOptionsAndSnapshot(t *testing.T) {
	apix.ResetRegistry()

	ref := &apix.RouteRef{
		Method:        apix.MethodPost,
		Path:          "/v1/items",
		Summary:       "create item",
		RequestType:   reflect.TypeOf(sampleReq{}),
		SuccessStatus: http.StatusCreated,
		Responses: map[int]*apix.ResponseRef{
			http.StatusCreated: {ModelType: reflect.TypeOf(sampleResp{})},
		},
	}

	apix.WithDescription("create a new item")(ref)
	apix.WithTags("items", "v1")(ref)
	apix.WithSuccessHeaders(http.StatusCreated, apix.HeaderRef{Name: "Location", SchemaType: "string", Required: true})(ref)
	apix.WithSuccessStatus(http.StatusCreated)(ref)
	apix.WithSecurity("BearerAuth")(ref)
	apix.WithRequestOverride(sampleReq{}, "application/json", map[string]any{"name": "widget"})(ref)
	apix.WithParameter(apix.Parameter{Name: "trace_id", In: "header", SchemaType: "string", Required: false, Description: "Tracing header"})(ref)
	apix.WithBodyRequired(true)(ref)
	apix.WithOperationID(apix.DefaultOperationID(apix.MethodPost, "/v1/items"))(ref)
	apix.WithSummary("create")(ref)
	apix.WithDescription("create item")(ref)
	apix.WithExplicitRequestModel(&sampleReq{}, "application/json")(ref)
	apix.WithDeprecated()(ref)

	apix.RegisterRoute(ref)

	snapshot := apix.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 route, got %d", len(snapshot))
	}

	item := snapshot[0]
	if item.SuccessStatus != http.StatusCreated {
		t.Fatalf("expected success status %d, got %d", http.StatusCreated, item.SuccessStatus)
	}
	if len(item.SuccessHeaders[http.StatusCreated]) != 1 {
		t.Fatalf("expected success header registered")
	}
	if len(item.Security) != 1 {
		t.Fatalf("expected security requirement registered")
	}
	if !item.BodyRequired {
		t.Fatalf("expected body to be required")
	}
	if len(item.Parameters) != 1 || item.Parameters[0].Name != "trace_id" {
		t.Fatalf("expected header parameter registered")
	}
	if item.Summary != "create" || item.Description != "create item" {
		t.Fatalf("expected summary/description options applied")
	}
	if item.ExplicitRequestModel == nil || item.RequestContentType != "application/json" {
		t.Fatalf("expected explicit request model override")
	}
	if !item.Deprecated {
		t.Fatalf("expected deprecated flag set")
	}

	opID := apix.DefaultOperationID(apix.MethodPost, "/v1/items")
	if item.OperationID != opID {
		t.Fatalf("expected operation id %q, got %q", opID, item.OperationID)
	}

	apix.ResetRegistry()
}

func TestEnsureResponse(t *testing.T) {
	ref := &apix.RouteRef{Responses: make(map[int]*apix.ResponseRef)}
	apix.EnsureResponse(ref, http.StatusOK, reflect.TypeOf(sampleResp{}))
	apix.EnsureResponse(ref, http.StatusOK, nil)
	resp := ref.Responses[http.StatusOK]
	if resp == nil || resp.ModelType != reflect.TypeOf(sampleResp{}) {
		t.Fatalf("response should retain original model type")
	}

	apix.WithDescriptionResponse("ok")(resp)
	apix.WithContentType("application/json")(resp)
	apix.WithExplicitModel(sampleResp{})(resp)
	apix.WithHeaders(apix.HeaderRef{Name: "X-Test", SchemaType: "string", Required: true})(resp)

	if resp.Description != "ok" || resp.ContentType != "application/json" {
		t.Fatalf("response options not applied")
	}
	if len(resp.Headers) != 1 {
		t.Fatalf("expected header applied")
	}
}
