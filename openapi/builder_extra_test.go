package openapi_test

import (
	"reflect"
	"testing"

	"github.com/Infra-Forge/apix"
	"github.com/Infra-Forge/apix/openapi"
)

type unionRequest struct {
	ID      *int           `json:"id,omitzero"`
	Payload map[string]any `json:"payload"`
}

func TestBuilderSchemaInferenceCornerCases(t *testing.T) {
	t.Cleanup(apix.ResetRegistry)

	ref := &apix.RouteRef{
		Method:      apix.MethodPut,
		Path:        "/api/union",
		RequestType: reflect.TypeOf(&unionRequest{}),
		Responses: map[int]*apix.ResponseRef{
			200: {ModelType: reflect.TypeOf(map[string]any{})},
		},
	}
	apix.RegisterRoute(ref)

	b := openapi.NewBuilder()
	doc, err := b.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	schema := doc.Components.Schemas
	if len(schema) == 0 {
		t.Fatalf("expected component schemas generated")
	}
}
