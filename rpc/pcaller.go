package rpc

import (
	"io"
	"net"
	"net/rpc"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
)

// PersistentCaller is a wrapper for session pooling and RPC calling.
type PersistentCaller struct {
	pool       NodeConnPool
	client     *Client
	TargetAddr string
	TargetID   proto.NodeID
	sync.Mutex
}

// NewPersistentCallerWithPool returns a persistent RPCCaller.
//  IMPORTANT: If a PersistentCaller is firstly used by a DHT.Ping, which is an anonymous
//  ETLS connection. It should not be used by any other RPC except DHT.Ping.
func NewPersistentCallerWithPool(pool NodeConnPool, target proto.NodeID) *PersistentCaller {
	return &PersistentCaller{
		pool:     pool,
		TargetID: target,
	}
}

func (c *PersistentCaller) initClient(isAnonymous bool) (err error) {
	c.Lock()
	defer c.Unlock()
	if c.client == nil {
		var conn net.Conn
		conn, err = DialToNodeWithPool(c.pool, c.TargetID, isAnonymous)
		if err != nil {
			err = errors.Wrap(err, "dial to node failed")
			return
		}
		//conn.SetDeadline(time.Time{})
		c.client = NewClientWithConn(conn)
	}
	return
}

// Call invokes the named function, waits for it to complete, and returns its error status.
func (c *PersistentCaller) Call(method string, args interface{}, reply interface{}) (err error) {
	startTime := time.Now()
	defer func() {
		recordRPCCost(startTime, method, err)
	}()

	err = c.initClient(method == route.DHTPing.String())
	if err != nil {
		err = errors.Wrap(err, "init PersistentCaller client failed")
		return
	}
	err = c.client.Call(method, args, reply)
	if err != nil {
		if err == io.EOF ||
			err == io.ErrUnexpectedEOF ||
			err == io.ErrClosedPipe ||
			err == rpc.ErrShutdown ||
			strings.Contains(strings.ToLower(err.Error()), "shut down") ||
			strings.Contains(strings.ToLower(err.Error()), "broken pipe") {
			// if got EOF, retry once
			reconnectErr := c.ResetClient(method)
			if reconnectErr != nil {
				err = errors.Wrap(reconnectErr, "reconnect failed")
			}
		}
		err = errors.Wrapf(err, "call %s failed", method)
		return
	}
	return
}

// ResetClient resets client.
func (c *PersistentCaller) ResetClient(method string) (err error) {
	c.Lock()
	c.Close()
	c.client = nil
	c.Unlock()
	return
}

// CloseStream closes the stream and RPC client.
func (c *PersistentCaller) CloseStream() {
	if c.client != nil {
		if c.client.Conn != nil {
			_ = c.client.Conn.Close()
		}
		c.client.Close()
	}
}

// Close closes the stream and RPC client.
func (c *PersistentCaller) Close() {
	c.CloseStream()
	//c.pool.Remove(c.TargetID)
}
