package api

import (
	"encoding/json"
	"net/http"
)

type httpconn struct {
	rw http.ResponseWriter
	r  *http.Request
}

// HTTPStream is data stream as jsonrpc2.ObjectStream over HTTP transport.
type HTTPStream struct {
	conn httpconn
}

// NewHTTPStream creates a new HTTPStream.
func NewHTTPStream(conn httpconn) HTTPStream {
	return HTTPStream{conn: conn}
}

// WriteObject implements jsonrpc2.ObjectStream.WriteObject.
func (t HTTPStream) WriteObject(obj interface{}) error {
	t.conn.rw.Header().Add("Content-Type", "application/json")
	return json.NewEncoder(t.conn.rw).Encode(obj)
}

// ReadObject implements jsonrpc2.ObjectStream.ReadObject.
func (t HTTPStream) ReadObject(v interface{}) error {
	return json.NewDecoder(t.conn.r.Body).Decode(v)
}

// Close implements jsonrpc2.ObjectStream.Close.
func (t HTTPStream) Close() error {
	return t.conn.r.Body.Close()
}
