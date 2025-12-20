package errorhandler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	apix "github.com/Infra-Forge/infra-apix"
)

// HandleError handles HTTP errors with support for StatusCoder interface
// and RFC 9457 Problem Details.
//
// This function is shared between chi and mux adapters to eliminate duplication.
func HandleError(w http.ResponseWriter, r *http.Request, err error, useProblemDetails bool) {
	// First check for StatusCoder interface (new pattern)
	var statusCoder apix.StatusCoder
	if errors.As(err, &statusCoder) {
		status := statusCoder.HTTPStatus()

		// If Problem Details is enabled, serialize as RFC 9457
		if useProblemDetails {
			problem := apix.ToProblemDetails(err)

			// Marshal first to check for errors before writing headers
			data, marshalErr := json.Marshal(problem)
			if marshalErr != nil {
				// Marshaling failed - fall back to plain error response
				http.Error(w, err.Error(), status)
				return
			}

			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(status)
			w.Write(data)
			return
		}

		http.Error(w, err.Error(), status)
		return
	}

	// Default to 500 for unrecognized errors
	// Log the original error server-side for debugging
	log.Printf("unrecognized error in %s %s: %v", r.Method, r.URL.Path, err)

	// Return generic message to client (don't leak internal error details)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
