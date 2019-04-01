package rpc

import (
	"context"
	"net"
	"net/rpc"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

// ServeDirect serves conn directly.
func ServeDirect(ctx context.Context, server *rpc.Server, conn net.Conn, remote proto.RawNodeID) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	nodeAwareCodec := NewNodeAwareServerCodec(ctx, utils.GetMsgPackServerCodec(conn), &remote)
	server.ServeCodec(nodeAwareCodec)
}
