package api

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

func init() {
	registerMethod("echo", echo)
}

func echo(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
	result interface{}, err error,
) {
	return req.Params, nil
}
