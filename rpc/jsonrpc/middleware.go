package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/sourcegraph/jsonrpc2"
)

// Validator is designed for params checking.
type Validator interface {
	Validate() error
}

// middleware: unmarshal req.Params(JSON array) to pre-defined structures (Object).
func processParams(h HandlerFunc, paramsType reflect.Type) HandlerFunc {
	return func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
		result interface{}, err error,
	) {
		paramsNew := reflect.New(paramsType)
		paramsElem := paramsNew.Elem()
		paramsArray := make([]interface{}, paramsElem.NumField())
		for i := 0; i < paramsElem.NumField(); i++ {
			paramsArray[i] = paramsElem.Field(i).Addr().Interface()
		}

		// Unmarshal JSON array to object
		// e.g. "[0,10]" --> struct { From: 0, To: 10 }
		if err := json.Unmarshal(*req.Params, &paramsArray); err != nil {
			return nil, err
		}

		if len(paramsArray) != paramsElem.NumField() {
			return nil, fmt.Errorf("unexpected parameters, expected %d but got %d",
				paramsElem.NumField(), len(paramsArray))
		}

		// parameters validator
		params := paramsNew.Interface()
		if t, ok := params.(Validator); ok {
			if err := t.Validate(); err != nil {
				return nil, &jsonrpc2.Error{
					Code:    jsonrpc2.CodeInvalidParams,
					Message: err.Error(),
				}
			}
		}

		ctx = context.WithValue(ctx, interface{}("_params"), params)
		return h(ctx, conn, req)
	}
}
