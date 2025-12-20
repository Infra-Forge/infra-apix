package errorhandler

import (
	"encoding/json"
	"errors"
	"net/http"

	apix "github.com/Infra-Forge/infra-apix"
)

// LegacyHTTPError represents the legacy httpError interface for backward compatibility.
// Adapters can implement this interface for their unexported httpError types.
type LegacyHTTPError interface {
	error
	// HTTPStatus returns the HTTP status code (for legacy errors that don't implement StatusCoder)
	HTTPStatus() int
	// Message returns the error message
	Message() string
}

// HandleError handles HTTP errors with support for StatusCoder interface,
// RFC 9457 Problem Details, and legacy httpError types.
//
// This function is shared between chi and mux adapters to eliminate duplication.
func HandleError(w http.ResponseWriter, err error, useProblemDetails bool) {
	// First check for StatusCoder interface (new pattern)
	var statusCoder apix.StatusCoder
	if errors.As(err, &statusCoder) {
		status := statusCoder.HTTPStatus()

		// If Problem Details is enabled, serialize as RFC 9457
		if useProblemDetails {
			problem := apix.ToProblemDetails(err)
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(problem)
			return
		}

		http.Error(w, err.Error(), status)
		return
	}

	// Default to 500 for unrecognized errors
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
