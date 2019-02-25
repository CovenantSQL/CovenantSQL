package jsonrpc_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/sourcegraph/jsonrpc2"
	wsstream "github.com/sourcegraph/jsonrpc2/websocket"

	"github.com/CovenantSQL/CovenantSQL/rpc/jsonrpc"
)

var (
	invalidHandler jsonrpc.HandlerFunc = nil

	echoHandler = func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
		result interface{}, err error,
	) {
		return req.Params, nil
	}

	incHandler = func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (
		result interface{}, err error,
	) {
		params := ctx.Value("_params").(*incPayload)
		return params.Number + 1, nil
	}
)

type incPayload struct {
	Number int `json:"number"`
}

func (p *incPayload) Validate() error {
	if p.Number < 0 {
		return errors.New("invalid number")
	}
	return nil
}

func TestAll(t *testing.T) {
	Convey("RegisterMethod", t, func() {
		var testCases = []struct {
			name        string
			method      string
			handlerFunc jsonrpc.HandlerFunc
			paramsType  interface{}
			willPanic   bool
		}{
			{
				name:        "register a nil HandlerFunc",
				method:      "nil",
				handlerFunc: invalidHandler,
			},
			{
				name:        "register method with nil params type",
				method:      "echo",
				handlerFunc: echoHandler,
			},
			{
				name:        "register method with concreate params type (elem)",
				method:      "inc",
				handlerFunc: incHandler,
				paramsType:  incPayload{},
			},
			{
				name:        "register method with concreate params type (pointer)",
				method:      "another_inc",
				handlerFunc: incHandler,
				paramsType:  new(incPayload),
			},
			{
				name:        "should panic on duplicated registration",
				method:      "echo",
				handlerFunc: echoHandler,
				willPanic:   true,
			},
		}

		for i, c := range testCases {
			Convey(fmt.Sprintf("case#%d: %s", i, c.name), FailureContinues, func() {
				opAssert := ShouldNotPanic
				if c.willPanic {
					opAssert = ShouldPanic
				}
				So(func() {
					jsonrpc.RegisterMethod(c.method, c.handlerFunc, c.paramsType)
				}, opAssert)
			})
		}
	})

	Convey("Websocket Serve Integration (handlers, middlewares)", t, func() {
		var server = &jsonrpc.WebsocketServer{
			Server: http.Server{
				Addr: ":999999",
			},
		}
		So(server.Serve(), ShouldBeError)

		server.Addr = ":8989"
		go server.Serve()

		// setup client
		client, err := setupWebsocketClient("ws://localhost" + server.Addr)
		So(err, ShouldBeNil)

		var testCases = []struct {
			name                 string
			method               string
			params               interface{}
			expectedError        error
			expectedResult       interface{}
			expectedResultAssert func(actual interface{}, expected ...interface{}) string
		}{
			{
				name:          "nil method should not be found",
				method:        "nil",
				params:        nil,
				expectedError: errors.New("nil pointer to handler"),
			},
			{
				name:          "unknown method should not be found",
				method:        "unknown",
				params:        nil,
				expectedError: errors.New("method not found"),
			},
			{
				name:           "echo method should work",
				method:         "echo",
				params:         "hello",
				expectedResult: "hello",
			},
			{
				name:                 "inc method should work",
				method:               "inc",
				params:               []interface{}{10},
				expectedResult:       11,
				expectedResultAssert: ShouldEqual,
			},
			{
				name:          "inc method should fail on invalid payload (unmarshal error)",
				method:        "inc",
				params:        []interface{}{"not a number"},
				expectedError: errors.New("unmarshal error"),
			},
			{
				name:          "inc method should fail on invalid payload (incorrect fields)",
				method:        "inc",
				params:        []interface{}{10, 11},
				expectedError: errors.New("unexpected parameters, expected 1 but got 2"),
			},
			{
				name:          "inc method should fail on invalid payload (validation error)",
				method:        "inc",
				params:        []interface{}{-1},
				expectedError: errors.New("invalid number"),
			},
		}

		for i, c := range testCases {
			Convey(fmt.Sprintf("case#%d: %s", i, c.name), FailureContinues, func() {
				var result interface{}
				err := client.Call(
					context.Background(),
					c.method,
					c.params,
					&result,
				)
				if c.expectedError != nil {
					So(err, ShouldBeError)
					t.Logf("Expected: %v, Actual: %v", c.expectedError, err)
				} else {
					So(err, ShouldBeNil)
					op := ShouldResemble
					if c.expectedResultAssert != nil {
						op = c.expectedResultAssert
					}
					So(c.expectedResult, op, result)
				}
			})
		}

		Reset(func() {
			server.Shutdown()
		})
	})
}

func setupWebsocketClient(addr string) (client *jsonrpc2.Conn, err error) {
	var dial = func(ctx context.Context, addr string) (client *jsonrpc2.Conn, err error) {
		conn, _, err := websocket.DefaultDialer.DialContext(
			context.Background(),
			addr,
			nil,
		)
		if err != nil {
			return nil, err
		}

		var connOpts []jsonrpc2.ConnOpt
		return jsonrpc2.NewConn(
			context.Background(),
			wsstream.NewObjectStream(conn),
			nil,
			connOpts...,
		), nil
	}

	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		client, err = dial(ctx, addr)
		if err == nil {
			break
		}
	}

	return client, err
}
