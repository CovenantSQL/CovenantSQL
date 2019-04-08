/*
 * Copyright 2018-2019 The CovenantSQL Authors.
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

package rpc

import (
	"io"
	"net/rpc"

	"github.com/CovenantSQL/CovenantSQL/utils"
)

// Client defines the RPC client interface.
type Client interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
	Go(serviceMethod string, args interface{}, reply interface{}, done chan *rpc.Call) *rpc.Call
	Close() error
}

// LastErrSetter defines the extend method to set client last error.
type LastErrSetter interface {
	SetLastErr(error)
}

// NewClient returns a new Client with stream.
//
// NOTE(leventeliu): ownership of stream is passed through:
//   io.Closer -> rpc.ClientCodec -> *rpc.Client
// Closing the *rpc.Client will cause io.Closer invoked.
func NewClient(stream io.ReadWriteCloser) (client *rpc.Client) {
	return rpc.NewClientWithCodec(utils.GetMsgPackClientCodec(stream))
}
