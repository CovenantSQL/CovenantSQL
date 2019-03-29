package rpc

import (
	"context"
	"expvar"
	"net/rpc"
	"sync"
	"time"

	"github.com/pkg/errors"
	mw "github.com/zserge/metric"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
)

var (
	callRPCExpvarLock sync.Mutex
)

func recordRPCCost(startTime time.Time, method string, err error) {
	var (
		name, nameC string
		val, valC   expvar.Var
	)
	costTime := time.Since(startTime)
	if err == nil {
		name = "t_succ:" + method
		nameC = "c_succ:" + method
	} else {
		name = "t_fail:" + method
		nameC = "c_fail:" + method
	}
	// Optimistically, val will not be nil except the first Call of method
	// expvar uses sync.Map
	// So, we try it first without lock
	val = expvar.Get(name)
	valC = expvar.Get(nameC)
	if val == nil || valC == nil {
		callRPCExpvarLock.Lock()
		val = expvar.Get(name)
		if val == nil {
			expvar.Publish(name, mw.NewHistogram("10s1s", "1m5s", "1h1m"))
			expvar.Publish(nameC, mw.NewCounter("10s1s", "1h1m"))
		}
		callRPCExpvarLock.Unlock()
		val = expvar.Get(name)
		valC = expvar.Get(nameC)
	}
	val.(mw.Metric).Add(costTime.Seconds())
	valC.(mw.Metric).Add(1)
	return
}

type Caller struct {
	pool NodeConnPool
}

func NewCaller(pool NodeConnPool) *Caller {
	return &Caller{
		pool: pool,
	}
}

func (c *Caller) CallNodeWithContext(
	ctx context.Context, node proto.NodeID, method string, args, reply interface{}) (err error,
) {
	startTime := time.Now()
	defer func() {
		recordRPCCost(startTime, method, err)
	}()

	conn, err := DialToNodeWithPool(c.pool, node, method == route.DHTPing.String())
	if err != nil {
		err = errors.Wrapf(err, "dial to node %s failed", node)
		return
	}
	defer conn.Close()

	client := NewClientWithConn(conn)
	defer client.Close()

	// TODO(xq262144): golang net/rpc does not support cancel in progress calls
	ch := client.Go(method, args, reply, make(chan *rpc.Call, 1))

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case call := <-ch.Done:
		err = call.Error
	}

	return
}

func (c *Caller) CallNode(node proto.NodeID, method string, args, reply interface{}) (err error) {
	return c.CallNodeWithContext(context.Background(), node, method, args, reply)
}
