/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mux

import (
	"context"
	"io"
	"net"
	nrpc "net/rpc"

	"github.com/pkg/errors"
	mux "github.com/xtaci/smux"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// ServeMux takes conn and serves as a multiplexing server.
func ServeMux(ctx context.Context, server *nrpc.Server, conn net.Conn, remoteNodeID proto.RawNodeID) {
	sess, err := mux.Server(conn, mux.DefaultConfig())
	if err != nil {
		err = errors.Wrap(err, "create mux server failed")
		return
	}
	defer func() { _ = sess.Close() }()

sessionLoop:
	for {
		select {
		case <-ctx.Done():
			log.Info("stopping Session Loop")
			break sessionLoop
		default:
			muxConn, err := sess.AcceptStream()
			if err != nil {
				if err == io.EOF {
					//log.WithField("remote", remoteNodeID).Debug("session connection closed")
				} else {
					err = errors.Wrapf(err, "session accept failed, remote: %s", remoteNodeID)
				}
				break sessionLoop
			}
			ctx, cancelFunc := context.WithCancel(context.Background())
			go func() {
				<-muxConn.GetDieCh()
				cancelFunc()
			}()
			nodeAwareCodec := rpc.NewNodeAwareServerCodec(ctx, utils.GetMsgPackServerCodec(muxConn), &remoteNodeID)
			go server.ServeCodec(nodeAwareCodec)
		}
	}
}
