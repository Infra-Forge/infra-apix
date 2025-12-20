package openapi_test

import (
	"net/http"
	"reflect"
	"testing"

	apix "github.com/Infra-Forge/infra-apix"
	"github.com/Infra-Forge/infra-apix/openapi"
	"github.com/getkin/kin-openapi/openapi3"
)

// Test model for file upload with multipart/form-data
type FileUploadRequest struct {
	File        []byte `json:"file" format:"binary" description:"File to upload"`
	Title       string `json:"title" description:"File title"`
	Description string `json:"description,omitempty" description:"Optional description"`
	Tags        string `json:"tags,omitempty" description:"Comma-separated tags"`
}

type FileUploadResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// Test model for form-urlencoded
type LoginForm struct {
	Username string `json:"username" description:"User login name"`
	Password string `json:"password" description:"User password"`
	Remember bool   `json:"remember,omitempty" description:"Remember me option"`
}

func TestMultipartFormDataContentType(t *testing.T) {
	apix.ResetRegistry()

	ref := &apix.RouteRef{
		Method:      apix.MethodPost,
		Path:        "/api/upload",
		Summary:     "Upload file",
		RequestType: reflect.TypeOf(FileUploadRequest{}),
		Responses: map[int]*apix.ResponseRef{
			http.StatusCreated: {ModelType: reflect.TypeOf(FileUploadResponse{})},
		},
	}

	apix.WithMultipartFormData()(ref)
	apix.RegisterRoute(ref)

	builder := openapi.NewBuilder()
	builder.Info.Title = "File Upload API"
	builder.Info.Version = "1.0.0"

	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify multipart/form-data content type
	pathItem := doc.Paths.Value("/api/upload")
	if pathItem == nil || pathItem.Post == nil {
		t.Fatal("POST /api/upload not found")
	}

	if pathItem.Post.RequestBody == nil || pathItem.Post.RequestBody.Value == nil {
		t.Fatal("Request body not found")
	}

	mediaType := pathItem.Post.RequestBody.Value.Content["multipart/form-data"]
	if mediaType == nil {
		t.Fatal("multipart/form-data media type not found")
	}

	// Verify schema has file field with binary format
	schema := mediaType.Schema
	if schema == nil || schema.Value == nil {
		t.Fatal("Schema not found")
	}

	// Check if schema is a reference to component
	var schemaValue *openapi3.Schema
	if schema.Ref != "" {
		// Schema is a reference, need to resolve it
		componentName := "openapi_test_FileUploadRequest"
		componentSchema := doc.Components.Schemas[componentName]
		if componentSchema == nil || componentSchema.Value == nil {
			t.Fatalf("Component schema %s not found", componentName)
		}
		schemaValue = componentSchema.Value
	} else {
		schemaValue = schema.Value
	}

	fileField := schemaValue.Properties["file"]
	if fileField == nil || fileField.Value == nil {
		t.Fatal("file field not found in schema")
	}

	if fileField.Value.Format != "binary" {
		t.Errorf("expected file field format 'binary', got %s", fileField.Value.Format)
	}

	if fileField.Value.Type == nil || !fileField.Value.Type.Is("string") {
		t.Errorf("expected file field type 'string', got %v", fileField.Value.Type)
	}
}

func TestFormURLEncodedContentType(t *testing.T) {
	apix.ResetRegistry()

	ref := &apix.RouteRef{
		Method:      apix.MethodPost,
		Path:        "/api/login",
		Summary:     "User login",
		RequestType: reflect.TypeOf(LoginForm{}),
		Responses: map[int]*apix.ResponseRef{
			http.StatusOK: {ModelType: nil},
		},
	}

	apix.WithFormURLEncoded()(ref)
	apix.RegisterRoute(ref)

	builder := openapi.NewBuilder()
	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify application/x-www-form-urlencoded content type
	pathItem := doc.Paths.Value("/api/login")
	if pathItem == nil || pathItem.Post == nil {
		t.Fatal("POST /api/login not found")
	}

	if pathItem.Post.RequestBody == nil || pathItem.Post.RequestBody.Value == nil {
		t.Fatal("Request body not found")
	}

	mediaType := pathItem.Post.RequestBody.Value.Content["application/x-www-form-urlencoded"]
	if mediaType == nil {
		t.Fatal("application/x-www-form-urlencoded media type not found")
	}

	// Verify schema exists
	if mediaType.Schema == nil {
		t.Fatal("Schema not found for form-urlencoded")
	}
}
