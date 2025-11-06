package fiber

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/Infra-Forge/apix"
	"github.com/gofiber/fiber/v3"
)

// RequestDecoder decodes the HTTP request body into dst, enforcing validation rules.
type RequestDecoder func(ctx context.Context, c fiber.Ctx, dst any) error

// ResponseEncoder writes the response payload using the provided status code.
type ResponseEncoder func(ctx context.Context, c fiber.Ctx, status int, payload any, ref *apix.RouteRef) error

// ErrorHandler handles errors from the typed handler.
type ErrorHandler func(ctx context.Context, c fiber.Ctx, err error) error

// Options configures adapter behaviour.
type Options struct {
	Decoder         RequestDecoder
	ResponseEncoder ResponseEncoder
	ErrorHandler    ErrorHandler
}

// FiberAdapter integrates apix route registration with fiber.App.
type FiberAdapter struct {
	app  *fiber.App
	opts Options
}

// New constructs a FiberAdapter with optional overrides.
func New(app *fiber.App, opts ...Options) *FiberAdapter {
	adapter := &FiberAdapter{app: app}
	if len(opts) > 0 {
		adapter.opts = opts[0]
	}
	return adapter
}

// App exposes the underlying fiber.App instance.
func (a *FiberAdapter) App() *fiber.App { return a.app }

// Register adds a handler for the provided method and path.
func Register[TReq any, TResp any](a *FiberAdapter, method apix.RouteMethod, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	if a == nil || a.app == nil {
		panic("apix/fiber: adapter not initialised")
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

	a.app.Add([]string{string(method)}, path, buildFiberHandler(a, handler, ref))
}

// Get registers a GET handler.
func Get[TResp any](a *FiberAdapter, path string, handler apix.HandlerFunc[apix.NoBody, TResp], opts ...apix.RouteOption) {
	Register[apix.NoBody, TResp](a, apix.MethodGet, path, handler, opts...)
}

// Post registers a POST handler.
func Post[TReq any, TResp any](a *FiberAdapter, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	Register[TReq, TResp](a, apix.MethodPost, path, handler, opts...)
}

// Put registers a PUT handler.
func Put[TReq any, TResp any](a *FiberAdapter, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	Register[TReq, TResp](a, apix.MethodPut, path, handler, opts...)
}

// Patch registers a PATCH handler.
func Patch[TReq any, TResp any](a *FiberAdapter, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	Register[TReq, TResp](a, apix.MethodPatch, path, handler, opts...)
}

// Delete registers a DELETE handler.
func Delete[TResp any](a *FiberAdapter, path string, handler apix.HandlerFunc[apix.NoBody, TResp], opts ...apix.RouteOption) {
	Register[apix.NoBody, TResp](a, apix.MethodDelete, path, handler, opts...)
}

func buildFiberHandler[TReq any, TResp any](a *FiberAdapter, handler apix.HandlerFunc[TReq, TResp], ref *apix.RouteRef) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := c.Context()

		var reqPtr *TReq
		if ref.RequestType != nil {
			reqVal := new(TReq)
			if err := a.decode(ctx, c, reqVal); err != nil {
				return a.handleError(ctx, c, err)
			}
			reqPtr = reqVal
		}

		resp, err := handler(ctx, reqPtr)
		if err != nil {
			return a.handleError(ctx, c, err)
		}

		if err := a.encode(ctx, c, ref.SuccessStatus, resp, ref); err != nil {
			return a.handleError(ctx, c, err)
		}
		return nil
	}
}

func (a *FiberAdapter) decode(ctx context.Context, c fiber.Ctx, dst any) error {
	if dec := a.opts.Decoder; dec != nil {
		return dec(ctx, c, dst)
	}
	return defaultDecoder(ctx, c, dst)
}

func (a *FiberAdapter) encode(ctx context.Context, c fiber.Ctx, status int, payload any, ref *apix.RouteRef) error {
	if enc := a.opts.ResponseEncoder; enc != nil {
		return enc(ctx, c, status, payload, ref)
	}
	return defaultEncoder(ctx, c, status, payload)
}

func (a *FiberAdapter) handleError(ctx context.Context, c fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}
	if handler := a.opts.ErrorHandler; handler != nil {
		return handler(ctx, c, err)
	}
	return defaultErrorHandler(ctx, c, err)
}

func defaultDecoder(ctx context.Context, c fiber.Ctx, dst any) error {
	body := c.Body()
	if len(body) == 0 {
		return &httpError{status: http.StatusBadRequest, message: "request body required"}
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
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
	return nil
}

func defaultEncoder(ctx context.Context, c fiber.Ctx, status int, payload any) error {
	if status == http.StatusNoContent || payload == nil || isNoBody(reflect.TypeOf(payload)) {
		return c.SendStatus(status)
	}
	return c.Status(status).JSON(payload)
}

func defaultErrorHandler(ctx context.Context, c fiber.Ctx, err error) error {
	var httpErr *httpError
	if errors.As(err, &httpErr) {
		return c.Status(httpErr.status).JSON(fiber.Map{"error": httpErr.message})
	}
	return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
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
