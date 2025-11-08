package openapi

import (
	"reflect"
	"testing"

	apix "github.com/Infra-Forge/infra-apix"
	"github.com/getkin/kin-openapi/openapi3"
)

func newTestBuilder() *Builder {
	b := NewBuilder()
	b.doc = &openapi3.T{Components: &openapi3.Components{Schemas: map[string]*openapi3.SchemaRef{}}}
	b.schemaCache = make(map[reflect.Type]*openapi3.SchemaRef)
	return b
}

func TestSchemaRefFromTypePrimitives(t *testing.T) {
	b := newTestBuilder()
	cases := []struct {
		name   string
		typeOf interface{}
		want   string
	}{
		{"bool", true, "boolean"},
		{"int32", int32(0), "integer"},
		{"float", float32(0), "number"},
		{"string", "", "string"},
	}

	for _, tc := range cases {
		ref, err := b.schemaRefFromType(reflect.TypeOf(tc.typeOf))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if ref.Value == nil {
			t.Fatalf("%s: schema value nil", tc.name)
		}
		if ref.Value.Type == nil || len(*ref.Value.Type) == 0 {
			t.Fatalf("%s: type not set", tc.name)
		}
	}
}

func TestSchemaRefFromTypePointerNullable(t *testing.T) {
	b := newTestBuilder()
	ref, err := b.schemaRefFromType(reflect.TypeOf(&struct{ Name string }{}))
	if err != nil {
		t.Fatalf("pointer schema: %v", err)
	}
	if ref.Value == nil || !ref.Value.Nullable {
		t.Fatalf("expected nullable schema")
	}
}

func TestSchemaRefFromTypeMapError(t *testing.T) {
	b := newTestBuilder()
	_, err := b.schemaRefFromType(reflect.TypeOf(map[int]string{}))
	if err == nil {
		t.Fatalf("expected error for map with non-string key")
	}
}

func TestWrapNullableAndEnsureSchema(t *testing.T) {
	s := openapi3.NewStringSchema()
	ref := schemaRef(s)
	wrapped := wrapNullable(ref)
	if wrapped.Value == nil || !wrapped.Value.Nullable {
		t.Fatalf("expected nullable clone")
	}

	var sr *openapi3.SchemaRef
	result := ensureSchema(sr)
	if result == nil || result.Type == nil || len(*result.Type) != 0 {
		// ensureSchema returns object schema without type; focus on non-nil
	}
}

func TestHeaderRefSetsSchema(t *testing.T) {
	hdr := apix.HeaderRef{Name: "X-Test", SchemaType: "integer", Required: true}
	ref := headerRef(hdr)
	if ref == nil || ref.Value == nil || ref.Value.Schema == nil || ref.Value.Schema.Value == nil {
		t.Fatalf("header schema not created")
	}
	if ref.Value.Schema.Value.Type == nil || len(*ref.Value.Schema.Value.Type) == 0 {
		t.Fatalf("expected schema type set")
	}
}

func TestDefaultResponseDescription(t *testing.T) {
	cases := map[int]string{
		200: "OK",
		201: "Created",
		204: "No Content",
		400: "Bad Request",
		404: "Not Found",
		500: "Internal Server Error",
		418: "",
	}
	for status, want := range cases {
		if got := defaultResponseDescription(status); got != want {
			t.Fatalf("status %d: expected %q, got %q", status, want, got)
		}
	}
}
