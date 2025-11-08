package openapi_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	apix "github.com/Infra-Forge/infra-apix"
	"github.com/Infra-Forge/infra-apix/openapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type DocumentModel struct {
	ID            uuid.UUID  `json:"id"`
	FileName      string     `json:"file_name"`
	ContentType   string     `json:"content_type"`
	FileSize      int64      `json:"file_size"`
	ProcessStatus string     `json:"process_status"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type TransactionModel struct {
	ID              uuid.UUID       `json:"id"`
	DocumentID      uuid.UUID       `json:"document_id"`
	TransactionDate time.Time       `json:"transaction_date"`
	Description     string          `json:"description"`
	Amount          decimal.Decimal `json:"amount"`
	TransactionType string          `json:"transaction_type"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type PaginationMeta struct {
	Page    int  `json:"page"`
	PerPage int  `json:"per_page"`
	Total   int  `json:"total"`
	Pages   int  `json:"pages"`
	HasNext bool `json:"has_next"`
	HasPrev bool `json:"has_prev"`
}

type DocumentListResponse struct {
	Data       []DocumentModel `json:"data"`
	Pagination PaginationMeta  `json:"pagination"`
}

type TransactionListResponse struct {
	Data       []TransactionModel `json:"data"`
	Pagination PaginationMeta     `json:"pagination"`
}

func TestNestedStructSchemas(t *testing.T) {
	apix.ResetRegistry()

	// Register a route with nested struct response
	apix.RegisterRoute(&apix.RouteRef{
		Method:  apix.MethodGet,
		Path:    "/api/documents",
		Summary: "List documents",
		Responses: map[int]*apix.ResponseRef{
			200: {ModelType: reflect.TypeOf(DocumentListResponse{})},
		},
	})

	builder := openapi.NewBuilder()
	builder.Info.Title = "Test API"
	builder.Info.Version = "1.0.0"

	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Validate that all schemas are present
	expectedSchemas := []string{
		"openapi_test_DocumentListResponse",
		"openapi_test_DocumentModel",
		"openapi_test_PaginationMeta",
	}

	for _, schemaName := range expectedSchemas {
		if doc.Components.Schemas[schemaName] == nil {
			t.Errorf("Missing schema: %s", schemaName)
			t.Logf("Available schemas: %v", getSchemaKeys(doc.Components.Schemas))
		}
	}

	// Validate the spec with proper context
	ctx := context.Background()
	if err := doc.Validate(ctx); err != nil {
		t.Fatalf("Spec validation failed: %v", err)
	}
}

func TestNestedStructWithDecimal(t *testing.T) {
	apix.ResetRegistry()

	// Register a route with decimal field
	apix.RegisterRoute(&apix.RouteRef{
		Method:  apix.MethodGet,
		Path:    "/api/transactions",
		Summary: "List transactions",
		Responses: map[int]*apix.ResponseRef{
			200: {ModelType: reflect.TypeOf(TransactionListResponse{})},
		},
	})

	builder := openapi.NewBuilder()
	builder.Info.Title = "Test API"
	builder.Info.Version = "1.0.0"

	doc, err := builder.Build(apix.Snapshot())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Validate the spec with proper context
	ctx := context.Background()
	if err := doc.Validate(ctx); err != nil {
		t.Fatalf("Spec validation failed: %v", err)
	}

	// Check that TransactionModel schema has decimal field
	txnSchema := doc.Components.Schemas["openapi_test_TransactionModel"]
	if txnSchema == nil || txnSchema.Value == nil {
		t.Fatal("TransactionModel schema not found")
	}

	amountProp := txnSchema.Value.Properties["amount"]
	if amountProp == nil || amountProp.Value == nil {
		t.Fatal("amount property not found")
	}

	if amountProp.Value.Format != "decimal" {
		t.Errorf("Expected decimal format, got %s", amountProp.Value.Format)
	}
}

func getSchemaKeys(schemas map[string]*openapi3.SchemaRef) []string {
	keys := make([]string, 0, len(schemas))
	for k := range schemas {
		keys = append(keys, k)
	}
	return keys
}
