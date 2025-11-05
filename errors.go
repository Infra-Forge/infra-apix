package apix

import (
	"net/http"
	"reflect"
)

// ErrorResponse is the standard error response schema for 4xx/5xx responses.
type ErrorResponse struct {
	Code    string `json:"code" description:"Error code identifier"`
	Message string `json:"message" description:"Human-readable error message"`
	Details any    `json:"details,omitempty" description:"Additional error details"`
}

var errorResponseType = reflect.TypeOf(ErrorResponse{})

// WithStandardErrors adds standard 4xx/5xx error responses to a route.
// This includes 400, 422, 500 with the shared ErrorResponse schema.
func WithStandardErrors() RouteOption {
	return func(r *RouteRef) {
		EnsureResponse(r, http.StatusBadRequest, errorResponseType,
			WithDescriptionResponse("Bad Request - Invalid input"))
		EnsureResponse(r, http.StatusUnprocessableEntity, errorResponseType,
			WithDescriptionResponse("Unprocessable Entity - Validation failed"))
		EnsureResponse(r, http.StatusInternalServerError, errorResponseType,
			WithDescriptionResponse("Internal Server Error"))
	}
}

// WithErrorResponse adds a specific error response with the standard ErrorResponse schema.
func WithErrorResponse(status int, description string) RouteOption {
	return func(r *RouteRef) {
		EnsureResponse(r, status, errorResponseType, WithDescriptionResponse(description))
	}
}

// WithCustomErrorResponse adds a custom error response with a specific model type.
func WithCustomErrorResponse(status int, model any, description string) RouteOption {
	return func(r *RouteRef) {
		modelType := typeOf(model)
		EnsureResponse(r, status, modelType, WithDescriptionResponse(description))
	}
}

// Common error response helpers

// WithBadRequestError adds a 400 Bad Request error response.
func WithBadRequestError(description string) RouteOption {
	if description == "" {
		description = "Bad Request - Invalid input"
	}
	return WithErrorResponse(http.StatusBadRequest, description)
}

// WithUnauthorizedError adds a 401 Unauthorized error response.
func WithUnauthorizedError(description string) RouteOption {
	if description == "" {
		description = "Unauthorized - Authentication required"
	}
	return WithErrorResponse(http.StatusUnauthorized, description)
}

// WithForbiddenError adds a 403 Forbidden error response.
func WithForbiddenError(description string) RouteOption {
	if description == "" {
		description = "Forbidden - Insufficient permissions"
	}
	return WithErrorResponse(http.StatusForbidden, description)
}

// WithNotFoundError adds a 404 Not Found error response.
func WithNotFoundError(description string) RouteOption {
	if description == "" {
		description = "Not Found - Resource does not exist"
	}
	return WithErrorResponse(http.StatusNotFound, description)
}

// WithConflictError adds a 409 Conflict error response.
func WithConflictError(description string) RouteOption {
	if description == "" {
		description = "Conflict - Resource already exists"
	}
	return WithErrorResponse(http.StatusConflict, description)
}

// WithValidationError adds a 422 Unprocessable Entity error response.
func WithValidationError(description string) RouteOption {
	if description == "" {
		description = "Unprocessable Entity - Validation failed"
	}
	return WithErrorResponse(http.StatusUnprocessableEntity, description)
}

// WithInternalServerError adds a 500 Internal Server Error response.
func WithInternalServerError(description string) RouteOption {
	if description == "" {
		description = "Internal Server Error"
	}
	return WithErrorResponse(http.StatusInternalServerError, description)
}

// WithServiceUnavailableError adds a 503 Service Unavailable error response.
func WithServiceUnavailableError(description string) RouteOption {
	if description == "" {
		description = "Service Unavailable - Temporary outage"
	}
	return WithErrorResponse(http.StatusServiceUnavailable, description)
}

