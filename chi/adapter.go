package chi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	apix "github.com/Infra-Forge/infra-apix"
	"github.com/go-chi/chi/v5"
)

// RequestDecoder decodes the HTTP request body into dst, enforcing validation rules.
type RequestDecoder func(ctx context.Context, w http.ResponseWriter, r *http.Request, dst any) error

// ResponseEncoder writes the response payload using the provided status code.
type ResponseEncoder func(ctx context.Context, w http.ResponseWriter, r *http.Request, status int, payload any, ref *apix.RouteRef) error

// ErrorHandler handles errors from the typed handler.
type ErrorHandler func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error)

// Validator validates request payloads.
type Validator interface {
	Validate(any) error
}

// Options configures adapter behaviour.
type Options struct {
	Decoder         RequestDecoder
	ResponseEncoder ResponseEncoder
	ErrorHandler    ErrorHandler
	Validator       Validator
}

// ChiAdapter integrates apix route registration with chi.Router.
type ChiAdapter struct {
	r    chi.Router
	opts Options
}

// New constructs a ChiAdapter with optional overrides.
func New(r chi.Router, opts ...Options) *ChiAdapter {
	adapter := &ChiAdapter{r: r}
	if len(opts) > 0 {
		adapter.opts = opts[0]
	}
	return adapter
}

// Router exposes the underlying chi.Router instance.
func (a *ChiAdapter) Router() chi.Router { return a.r }

// Register adds a handler for the provided method and path.
func Register[TReq any, TResp any](a *ChiAdapter, method apix.RouteMethod, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	if a == nil || a.r == nil {
		panic("apix/chi: adapter not initialised")
	}

	ref := &apix.RouteRef{
		Method:         method,
		Path:           path,
		Responses:      make(map[int]*apix.ResponseRef),
		SuccessHeaders: make(map[int][]apix.HeaderRef),
		HandlerType:    reflect.TypeOf(handler),
	}

	reqType := typeOf[TReq]()
	if reqType != nil && !isNoBody(reqType) {
		ref.RequestType = reqType
		if ref.RequestContentType == "" {
			ref.RequestContentType = "application/json"
		}
	}

	for _, opt := range opts {
		opt(ref)
	}

	if ref.SuccessStatus == 0 {
		ref.SuccessStatus = apix.DefaultSuccessStatus(method)
	}

	respType := typeOf[TResp]()
	if respType == nil {
		respType = reflect.TypeOf(struct{}{})
	}
	apix.EnsureResponse(ref, ref.SuccessStatus, respType)

	if ref.OperationID == "" {
		ref.OperationID = apix.DefaultOperationID(ref.Method, ref.Path)
	}

	apix.RegisterRoute(ref)

	a.r.Method(string(method), path, buildChiHandler(a, handler, ref))
}

// Get registers a GET handler.
func Get[TResp any](a *ChiAdapter, path string, handler apix.HandlerFunc[apix.NoBody, TResp], opts ...apix.RouteOption) {
	Register[apix.NoBody, TResp](a, apix.MethodGet, path, handler, opts...)
}

// Post registers a POST handler.
func Post[TReq any, TResp any](a *ChiAdapter, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	Register[TReq, TResp](a, apix.MethodPost, path, handler, opts...)
}

// Put registers a PUT handler.
func Put[TReq any, TResp any](a *ChiAdapter, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	Register[TReq, TResp](a, apix.MethodPut, path, handler, opts...)
}

// Patch registers a PATCH handler.
func Patch[TReq any, TResp any](a *ChiAdapter, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	Register[TReq, TResp](a, apix.MethodPatch, path, handler, opts...)
}

// Delete registers a DELETE handler.
func Delete[TResp any](a *ChiAdapter, path string, handler apix.HandlerFunc[apix.NoBody, TResp], opts ...apix.RouteOption) {
	Register[apix.NoBody, TResp](a, apix.MethodDelete, path, handler, opts...)
}

func buildChiHandler[TReq any, TResp any](a *ChiAdapter, handler apix.HandlerFunc[TReq, TResp], ref *apix.RouteRef) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var reqPtr *TReq
		if ref.RequestType != nil {
			reqVal := new(TReq)
			if err := a.decode(ctx, w, r, reqVal); err != nil {
				a.handleError(ctx, w, r, err)
				return
			}
			reqPtr = reqVal
		}

		resp, err := handler(ctx, reqPtr)
		if err != nil {
			a.handleError(ctx, w, r, err)
			return
		}

		if err := a.encode(ctx, w, r, ref.SuccessStatus, resp, ref); err != nil {
			a.handleError(ctx, w, r, err)
		}
	}
}

func (a *ChiAdapter) decode(ctx context.Context, w http.ResponseWriter, r *http.Request, dst any) error {
	if dec := a.opts.Decoder; dec != nil {
		return dec(ctx, w, r, dst)
	}
	return defaultDecoder(ctx, w, r, dst, a.opts.Validator)
}

func (a *ChiAdapter) encode(ctx context.Context, w http.ResponseWriter, r *http.Request, status int, payload any, ref *apix.RouteRef) error {
	if enc := a.opts.ResponseEncoder; enc != nil {
		return enc(ctx, w, r, status, payload, ref)
	}
	return defaultEncoder(ctx, w, r, status, payload)
}

func (a *ChiAdapter) handleError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	if handler := a.opts.ErrorHandler; handler != nil {
		handler(ctx, w, r, err)
		return
	}
	defaultErrorHandler(ctx, w, r, err)
}

func defaultDecoder(ctx context.Context, w http.ResponseWriter, r *http.Request, dst any, validator Validator) error {
	if r.Body == nil {
		return &httpError{status: http.StatusBadRequest, message: "request body required"}
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return &httpError{status: http.StatusBadRequest, message: "request body required"}
		}
		return &httpError{status: http.StatusBadRequest, message: err.Error()}
	}
	if decoder.More() {
		return &httpError{status: http.StatusBadRequest, message: "unexpected additional JSON content"}
	}
	if validator != nil {
		if err := validator.Validate(dst); err != nil {
			return &httpError{status: http.StatusUnprocessableEntity, message: err.Error()}
		}
	}
	return nil
}

func defaultEncoder(ctx context.Context, w http.ResponseWriter, r *http.Request, status int, payload any) error {
	if status == http.StatusNoContent || payload == nil || isNoBody(reflect.TypeOf(payload)) {
		w.WriteHeader(status)
		return nil
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}

func defaultErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	var httpErr *httpError
	if errors.As(err, &httpErr) {
		http.Error(w, httpErr.message, httpErr.status)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

type httpError struct {
	status  int
	message string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("http %d: %s", e.status, e.message)
}

func typeOf[T any]() reflect.Type {
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

var noBodyType = reflect.TypeOf(apix.NoBody{})

func isNoBody(t reflect.Type) bool {
	if t == nil {
		return true
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t == noBodyType
}
