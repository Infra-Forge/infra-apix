package echo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"

	"github.com/Infra-Forge/apix"
	"github.com/labstack/echo/v4"
)

// RequestDecoder decodes the HTTP request body into dst, enforcing validation rules.
type RequestDecoder func(ctx context.Context, c echo.Context, dst any) error

// ResponseEncoder writes the response payload using the provided status code.
type ResponseEncoder func(ctx context.Context, c echo.Context, status int, payload any, ref *apix.RouteRef) error

// ErrorTransformer transforms handler errors into Echo-compatible errors.
type ErrorTransformer func(err error) error

// Options configures adapter behaviour.
type Options struct {
	Decoder         RequestDecoder
	ResponseEncoder ResponseEncoder
	ErrorHandler    ErrorTransformer
}

// EchoAdapter integrates apix route registration with echo.Echo.
type EchoAdapter struct {
	e    *echo.Echo
	opts Options
}

// New constructs an EchoAdapter with optional overrides.
func New(e *echo.Echo, opts ...Options) *EchoAdapter {
	adapter := &EchoAdapter{e: e}
	if len(opts) > 0 {
		adapter.opts = opts[0]
	}
	return adapter
}

// Echo exposes the underlying echo.Echo instance.
func (a *EchoAdapter) Echo() *echo.Echo { return a.e }

// Register adds a handler for the provided method and path.
func Register[TReq any, TResp any](a *EchoAdapter, method apix.RouteMethod, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	if a == nil || a.e == nil {
		panic("apix/echo: adapter not initialised")
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

	a.e.Add(string(method), path, buildEchoHandler(a, handler, ref))
}

// Get registers a GET handler.
func Get[TResp any](a *EchoAdapter, path string, handler apix.HandlerFunc[apix.NoBody, TResp], opts ...apix.RouteOption) {
	Register[apix.NoBody, TResp](a, apix.MethodGet, path, handler, opts...)
}

// Post registers a POST handler.
func Post[TReq any, TResp any](a *EchoAdapter, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	Register[TReq, TResp](a, apix.MethodPost, path, handler, opts...)
}

// Put registers a PUT handler.
func Put[TReq any, TResp any](a *EchoAdapter, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	Register[TReq, TResp](a, apix.MethodPut, path, handler, opts...)
}

// Patch registers a PATCH handler.
func Patch[TReq any, TResp any](a *EchoAdapter, path string, handler apix.HandlerFunc[TReq, TResp], opts ...apix.RouteOption) {
	Register[TReq, TResp](a, apix.MethodPatch, path, handler, opts...)
}

// Delete registers a DELETE handler.
func Delete[TResp any](a *EchoAdapter, path string, handler apix.HandlerFunc[apix.NoBody, TResp], opts ...apix.RouteOption) {
	Register[apix.NoBody, TResp](a, apix.MethodDelete, path, handler, opts...)
}

func buildEchoHandler[TReq any, TResp any](a *EchoAdapter, handler apix.HandlerFunc[TReq, TResp], ref *apix.RouteRef) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		var reqPtr *TReq
		if ref.RequestType != nil {
			reqVal := new(TReq)
			if err := a.decode(ctx, c, reqVal); err != nil {
				return a.transformError(err)
			}
			reqPtr = reqVal
		}

		resp, err := handler(ctx, reqPtr)
		if err != nil {
			return a.transformError(err)
		}

		return a.encode(ctx, c, ref.SuccessStatus, resp, ref)
	}
}

func (a *EchoAdapter) decode(ctx context.Context, c echo.Context, dst any) error {
	if dec := a.opts.Decoder; dec != nil {
		return dec(ctx, c, dst)
	}
	return defaultDecoder(ctx, c, dst)
}

func (a *EchoAdapter) encode(ctx context.Context, c echo.Context, status int, payload any, ref *apix.RouteRef) error {
	if enc := a.opts.ResponseEncoder; enc != nil {
		return enc(ctx, c, status, payload, ref)
	}
	return defaultEncoder(ctx, c, status, payload)
}

func (a *EchoAdapter) transformError(err error) error {
	if err == nil {
		return nil
	}
	if tr := a.opts.ErrorHandler; tr != nil {
		return tr(err)
	}
	return err
}

func defaultDecoder(ctx context.Context, c echo.Context, dst any) error {
	req := c.Request()
	if req.Body == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "request body required")
	}
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return echo.NewHTTPError(http.StatusBadRequest, "request body required")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if decoder.More() {
		return echo.NewHTTPError(http.StatusBadRequest, "unexpected additional JSON content")
	}
	if v := c.Echo().Validator; v != nil {
		if err := v.Validate(dst); err != nil {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}
	}
	return nil
}

func defaultEncoder(ctx context.Context, c echo.Context, status int, payload any) error {
	if status == http.StatusNoContent || payload == nil || isNoBody(reflect.TypeOf(payload)) {
		return c.NoContent(status)
	}
	return c.JSON(status, payload)
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
