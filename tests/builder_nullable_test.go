package openapi_test

import (
	"reflect"
	"testing"

	"github.com/Infra-Forge/apix"
	"github.com/Infra-Forge/apix/openapi"
)

type nullableStruct struct {
	Optional *string `json:"optional,omitempty"`
	Numbers  []int   `json:"numbers"`
}

func TestBuildWithNullableStruct(t *testing.T) {
	t.Cleanup(apix.ResetRegistry)
	apix.RegisterRoute(&apix.RouteRef{
		Method:      apix.MethodPost,
		Path:        "/nullable",
		RequestType: reflect.TypeOf(nullableStruct{}),
		Responses:   map[int]*apix.ResponseRef{200: {}},
	})

	builder := openapi.NewBuilder()
	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if len(doc.Components.Schemas) == 0 {
		t.Fatalf("expected component schemas")
	}
}
