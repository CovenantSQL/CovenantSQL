package jsonrpc

import (
	"context"
	"fmt"
	"reflect"

	"github.com/sourcegraph/jsonrpc2"
)

var (
	defaultHandler = NewHandler()
)

// HandlerFunc is a function adapter to Handler.
type HandlerFunc func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (interface{}, error)

// RegisterMethod register a method to the default handler.
func RegisterMethod(method string, handlerFunc HandlerFunc, paramsType interface{}) {
	defaultHandler.RegisterMethod(method, handlerFunc, paramsType)
}

// Handler is a handler handling JSON-RPC protocol.
type Handler struct {
	methods map[string]HandlerFunc
}

// NewHandler creates a new JSONRPCHandler.
func NewHandler() *Handler {
	return &Handler{
		methods: make(map[string]HandlerFunc),
	}
}

// RegisterMethod register a method.
func (h *Handler) RegisterMethod(method string, handlerFunc HandlerFunc, paramsType interface{}) {
	if _, ok := h.methods[method]; ok {
		panic(fmt.Sprintf("method %q already registered", method))
	}

	if paramsType == nil {
		h.methods[method] = handlerFunc
		return
	}

	// Pre-process rpc parameters with a middleware
	typ := reflect.TypeOf(paramsType)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	h.methods[method] = processParams(handlerFunc, typ)
}

// Handle implements jsonrpc2.Handler.
func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	jsonrpc2.HandlerWithError(h.handle).Handle(ctx, conn, req)
}

// handle is a function to be used by jsonrpc2.Handler.
func (h *Handler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	defer func() {
		if p := recover(); p != nil {
			switch p := p.(type) {
			case error:
				err = p
			default:
				err = fmt.Errorf("%v", p)
			}
		}
	}()

	fn := h.methods[req.Method]
	if fn == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound}
	} else if req.Params == nil {
		// pre-check req.Params not be nil
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	return fn(ctx, conn, req)
}
