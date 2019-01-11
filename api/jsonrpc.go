package api

import (
	"context"
	"fmt"
	"reflect"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
)

var (
	jsonrpcHandler = NewJSONRPCHandler()
)

type jsonrpcHandlerFunc func(context.Context, *jsonrpc2.Conn, *jsonrpc2.Request) (interface{}, error)

func registerMethod(method string, handlerFunc jsonrpcHandlerFunc, paramsType interface{}) {
	log.WithField("method", method).Debug("api: register rpc method")

	if paramsType == nil {
		jsonrpcHandler.RegisterMethod(method, handlerFunc)
		return
	}

	// use a middleware component to pre-process params
	typ := reflect.TypeOf(paramsType)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	jsonrpcHandler.RegisterMethod(method, processParams(handlerFunc, typ))
}

// JSONRPCHandler is a handler handling JSON-RPC protocol.
type JSONRPCHandler struct {
	methods map[string]jsonrpcHandlerFunc
}

// NewJSONRPCHandler creates a new JSONRPCHandler.
func NewJSONRPCHandler() *JSONRPCHandler {
	return &JSONRPCHandler{
		methods: make(map[string]jsonrpcHandlerFunc),
	}
}

// RegisterMethod register a method.
func (h *JSONRPCHandler) RegisterMethod(method string, handlerFunc jsonrpcHandlerFunc) {
	h.methods[method] = handlerFunc
}

// Handler returns a jsonrpc2.Handler.
func (h *JSONRPCHandler) Handler() jsonrpc2.Handler {
	return jsonrpc2.HandlerWithError(h.handle)
}

var methodNotFound = func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	return nil, errors.Errorf("method not found: %q", req.Method)
}

// Handle implements jsonrpc2.Handler.
func (h *JSONRPCHandler) handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
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
		fn = methodNotFound
	} else if req.Params == nil {
		// pre-check req.Params not be nil
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	return fn(ctx, conn, req)
}
