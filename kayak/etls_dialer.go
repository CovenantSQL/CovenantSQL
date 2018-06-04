/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kayak

import (
	"net"
	"context"
)

// ETLSDialer defines connection resolution feature of kayak
type ETLSDialer struct {
}

// Dial implements Dialer.Dial
func (ed *ETLSDialer) Dial(serverID ServerID) (net.Conn, error) {
	return ed.DialContext(context.Background(), serverID)
}

// DialContext implements Dialer.DialContext
func (ed *ETLSDialer) DialContext(ctx context.Context, serverID ServerID) (net.Conn, error) {
	// TODO, current not implemented, , wait for node resolution and etls integration
	return nil, nil
}

var (
	_ Dialer = &ETLSDialer{}
)
