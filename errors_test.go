package apix_test

import (
	"net/http"
	"testing"

	"github.com/Infra-Forge/apix"
)

func TestErrorResponseType(t *testing.T) {
	err := apix.ErrorResponse{
		Code:    "VALIDATION_ERROR",
		Message: "Invalid input",
		Details: map[string]string{"field": "name"},
	}

	if err.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected code to be set")
	}
	if err.Message != "Invalid input" {
		t.Fatalf("expected message to be set")
	}
	if err.Details == nil {
		t.Fatalf("expected details to be set")
	}
}

func TestWithStandardErrors(t *testing.T) {
	apix.ResetRegistry()

	ref := &apix.RouteRef{
		Method:    apix.MethodPost,
		Path:      "/api/items",
		Responses: make(map[int]*apix.ResponseRef),
	}

	apix.WithStandardErrors()(ref)

	if len(ref.Responses) != 3 {
		t.Fatalf("expected 3 standard error responses, got %d", len(ref.Responses))
	}

	if ref.Responses[http.StatusBadRequest] == nil {
		t.Fatalf("expected 400 Bad Request response")
	}
	if ref.Responses[http.StatusUnprocessableEntity] == nil {
		t.Fatalf("expected 422 Unprocessable Entity response")
	}
	if ref.Responses[http.StatusInternalServerError] == nil {
		t.Fatalf("expected 500 Internal Server Error response")
	}

	// Verify descriptions
	if ref.Responses[http.StatusBadRequest].Description != "Bad Request - Invalid input" {
		t.Fatalf("unexpected 400 description")
	}
	if ref.Responses[http.StatusUnprocessableEntity].Description != "Unprocessable Entity - Validation failed" {
		t.Fatalf("unexpected 422 description")
	}
	if ref.Responses[http.StatusInternalServerError].Description != "Internal Server Error" {
		t.Fatalf("unexpected 500 description")
	}
}

func TestWithErrorResponse(t *testing.T) {
	apix.ResetRegistry()

	ref := &apix.RouteRef{
		Method:    apix.MethodGet,
		Path:      "/api/items",
		Responses: make(map[int]*apix.ResponseRef),
	}

	apix.WithErrorResponse(http.StatusNotFound, "Item not found")(ref)

	if ref.Responses[http.StatusNotFound] == nil {
		t.Fatalf("expected 404 response")
	}
	if ref.Responses[http.StatusNotFound].Description != "Item not found" {
		t.Fatalf("unexpected description")
	}
}

func TestWithCustomErrorResponse(t *testing.T) {
	apix.ResetRegistry()

	type CustomError struct {
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}

	ref := &apix.RouteRef{
		Method:    apix.MethodPost,
		Path:      "/api/custom",
		Responses: make(map[int]*apix.ResponseRef),
	}

	apix.WithCustomErrorResponse(http.StatusBadRequest, CustomError{}, "Custom error format")(ref)

	if ref.Responses[http.StatusBadRequest] == nil {
		t.Fatalf("expected 400 response")
	}
	if ref.Responses[http.StatusBadRequest].Description != "Custom error format" {
		t.Fatalf("unexpected description")
	}
	// Verify it's using the custom type, not ErrorResponse
	if ref.Responses[http.StatusBadRequest].ModelType.Name() != "CustomError" {
		t.Fatalf("expected CustomError type, got %s", ref.Responses[http.StatusBadRequest].ModelType.Name())
	}
}

func TestErrorHelpers(t *testing.T) {
	apix.ResetRegistry()

	ref := &apix.RouteRef{
		Method:    apix.MethodPost,
		Path:      "/api/test",
		Responses: make(map[int]*apix.ResponseRef),
	}

	// Test all helper functions
	apix.WithBadRequestError("")(ref)
	apix.WithUnauthorizedError("")(ref)
	apix.WithForbiddenError("")(ref)
	apix.WithNotFoundError("")(ref)
	apix.WithConflictError("")(ref)
	apix.WithValidationError("")(ref)
	apix.WithInternalServerError("")(ref)
	apix.WithServiceUnavailableError("")(ref)

	expectedStatuses := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusUnprocessableEntity,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}

	if len(ref.Responses) != len(expectedStatuses) {
		t.Fatalf("expected %d error responses, got %d", len(expectedStatuses), len(ref.Responses))
	}

	for _, status := range expectedStatuses {
		if ref.Responses[status] == nil {
			t.Fatalf("expected %d response", status)
		}
		if ref.Responses[status].Description == "" {
			t.Fatalf("expected description for %d", status)
		}
	}
}

func TestErrorHelpersWithCustomDescriptions(t *testing.T) {
	apix.ResetRegistry()

	ref := &apix.RouteRef{
		Method:    apix.MethodPost,
		Path:      "/api/test",
		Responses: make(map[int]*apix.ResponseRef),
	}

	apix.WithBadRequestError("Custom bad request")(ref)
	apix.WithNotFoundError("Custom not found")(ref)

	if ref.Responses[http.StatusBadRequest].Description != "Custom bad request" {
		t.Fatalf("expected custom description for 400")
	}
	if ref.Responses[http.StatusNotFound].Description != "Custom not found" {
		t.Fatalf("expected custom description for 404")
	}
}

func TestErrorResponsesDoNotOverwriteExisting(t *testing.T) {
	apix.ResetRegistry()

	ref := &apix.RouteRef{
		Method:    apix.MethodPost,
		Path:      "/api/test",
		Responses: make(map[int]*apix.ResponseRef),
	}

	// Add a custom 400 response first
	type CustomBadRequest struct {
		Reason string `json:"reason"`
	}
	apix.WithCustomErrorResponse(http.StatusBadRequest, CustomBadRequest{}, "Custom 400")(ref)

	// Try to add standard errors (should not overwrite existing 400)
	apix.WithStandardErrors()(ref)

	if ref.Responses[http.StatusBadRequest].ModelType.Name() != "CustomBadRequest" {
		t.Fatalf("expected custom 400 response to be preserved")
	}
	if ref.Responses[http.StatusBadRequest].Description != "Custom 400" {
		t.Fatalf("expected custom 400 description to be preserved")
	}

	// But 422 and 500 should be added
	if ref.Responses[http.StatusUnprocessableEntity] == nil {
		t.Fatalf("expected 422 to be added")
	}
	if ref.Responses[http.StatusInternalServerError] == nil {
		t.Fatalf("expected 500 to be added")
	}
}

func TestCombiningStandardErrorsWithSpecificErrors(t *testing.T) {
	apix.ResetRegistry()

	ref := &apix.RouteRef{
		Method:    apix.MethodGet,
		Path:      "/api/items/:id",
		Responses: make(map[int]*apix.ResponseRef),
	}

	// Add standard errors
	apix.WithStandardErrors()(ref)
	// Add specific errors
	apix.WithNotFoundError("Item not found")(ref)
	apix.WithUnauthorizedError("Authentication required")(ref)

	// Should have 400, 401, 404, 422, 500
	expectedCount := 5
	if len(ref.Responses) != expectedCount {
		t.Fatalf("expected %d responses, got %d", expectedCount, len(ref.Responses))
	}

	if ref.Responses[http.StatusNotFound].Description != "Item not found" {
		t.Fatalf("expected custom 404 description")
	}
	if ref.Responses[http.StatusUnauthorized].Description != "Authentication required" {
		t.Fatalf("expected custom 401 description")
	}
}
