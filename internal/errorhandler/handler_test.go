package errorhandler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apix "github.com/Infra-Forge/infra-apix"
)

func TestHandleErrorWithStatusCoder(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	err := apix.NotFound("resource not found")

	HandleError(w, r, err, false)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "resource not found") {
		t.Errorf("expected error message in body, got %s", w.Body.String())
	}
}

func TestHandleErrorWithProblemDetails(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	err := apix.BadRequest("invalid input")

	HandleError(w, r, err, true)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/problem+json" {
		t.Errorf("expected Content-Type application/problem+json, got %s", contentType)
	}
	if !strings.Contains(w.Body.String(), "invalid input") {
		t.Errorf("expected error message in body, got %s", w.Body.String())
	}
}

func TestHandleErrorWithGenericError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	err := errors.New("generic error")

	HandleError(w, r, err, false)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	// Should return generic message, not the actual error
	if !strings.Contains(w.Body.String(), "Internal Server Error") {
		t.Errorf("expected generic error message in body, got %s", w.Body.String())
	}
	// Should NOT leak the internal error message
	if strings.Contains(w.Body.String(), "generic error") {
		t.Errorf("should not leak internal error message, got %s", w.Body.String())
	}
}

func TestHandleErrorWithWrappedStatusCoder(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	innerErr := apix.Unauthorized("not authorized")
	wrappedErr := errors.Join(errors.New("wrapper"), innerErr)

	HandleError(w, r, wrappedErr, false)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestHandleErrorProblemDetailsWithExtensions(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	problem := apix.NewProblemDetails(http.StatusTeapot, "I'm a teapot", "Short and stout").
		WithExtension("spout", "handle")

	HandleError(w, r, problem, true)

	if w.Code != http.StatusTeapot {
		t.Errorf("expected status 418, got %d", w.Code)
	}
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/problem+json" {
		t.Errorf("expected Content-Type application/problem+json, got %s", contentType)
	}
	if !strings.Contains(w.Body.String(), "spout") {
		t.Errorf("expected extension field in body, got %s", w.Body.String())
	}
}
