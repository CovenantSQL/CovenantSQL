/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package proto

import (
	"time"
)

// NodeID is node name, will be generated from Hash(nodePublicKey)
type NodeID string

// NodeKey is node key on consistent hash ring, generate from Hash(NodeID)
type NodeKey uint64

// Node is all node info struct
type Node struct {
	Name      string
	Port      uint16
	Protocol  string
	ID        NodeID
	PublicKey string
}

// Envelope is the protocol
type Envelope struct {
	Version string
	TTL     time.Duration
	Expire  time.Duration
}

// PingReq is Ping RPC request
type PingReq struct {
	Node    Node
	Version string
	Envelope
}

// PingResp is Ping RPC response, i.e. Pong
type PingResp struct {
	Msg     string
	Version string
	Envelope
}

// FindValueReq is FindValue RPC request
type FindValueReq struct {
	NodeID  NodeID
	Count   int
	Version string
	Envelope
}

// FindValueResp is FindValue RPC response
type FindValueResp struct {
	Nodes   []Node
	Msg     string
	Version string
	Envelope
}
