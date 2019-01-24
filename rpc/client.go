/*
 * Copyright 2018 The CovenantSQL Authors.
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

// Package rpc provides RPC Client/Server functions
package rpc

import (
	"net"
	"net/rpc"

	"github.com/CovenantSQL/CovenantSQL/crypto/etls"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/pkg/errors"
	mux "github.com/xtaci/smux"
)

const (
	// ETLSHeaderSize is the header size with ETLSHeader + NodeID + Nonce
	ETLSHeaderSize = 2 + hash.HashBSize + 32
)

// Client is RPC client
type Client struct {
	*rpc.Client
	RemoteAddr string
	Conn       net.Conn
}

var (
	// MuxConfig holds the default mux config
	MuxConfig *mux.Config
	// DefaultDialer holds the default dialer of SessionPool
	DefaultDialer func(nodeID proto.NodeID) (conn net.Conn, err error)
)

func init() {
	MuxConfig = mux.DefaultConfig()
	DefaultDialer = dialToNode
}

// dial connects to a address with a Cipher
// address should be in the form of host:port
func dial(network, address string, remoteNodeID *proto.RawNodeID, cipher *etls.Cipher, isAnonymous bool) (c *etls.CryptoConn, err error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		err = errors.Wrapf(err, "connect to node %s failed", address)
		return
	}
	writeBuf := make([]byte, ETLSHeaderSize)
	writeBuf[0] = etls.ETLSMagicBytes[0]
	writeBuf[1] = etls.ETLSMagicBytes[1]
	if isAnonymous {
		copy(writeBuf[2:], kms.AnonymousRawNodeID.AsBytes())
		copy(writeBuf[2+hash.HashSize:], (&cpuminer.Uint256{}).Bytes())
	} else {
		// send NodeID + Uint256 Nonce
		var nodeIDBytes []byte
		var nonce *cpuminer.Uint256
		nodeIDBytes, err = kms.GetLocalNodeIDBytes()
		if err != nil {
			err = errors.Wrap(err, "get local node id failed")
			return
		}
		nonce, err = kms.GetLocalNonce()
		if err != nil {
			err = errors.Wrap(err, "get local nonce failed")
			return
		}
		copy(writeBuf[2:2+hash.HashSize], nodeIDBytes)
		copy(writeBuf[2+hash.HashSize:], nonce.Bytes())
	}
	wrote, err := conn.Write(writeBuf)
	if err != nil {
		err = errors.Wrap(err, "write node id and nonce failed")
		return
	}

	if wrote != ETLSHeaderSize {
		err = errors.Errorf("write header size not match %d", wrote)
		return
	}

	c = etls.NewConn(conn, cipher, remoteNodeID)
	return
}

// DialToNode ties use connection in pool, if fails then connects to the node with nodeID
func DialToNode(nodeID proto.NodeID, pool *SessionPool, isAnonymous bool) (conn net.Conn, err error) {
	if pool == nil || isAnonymous {
		var ETLSConn net.Conn
		var sess *mux.Session
		ETLSConn, err = dialToNodeEx(nodeID, isAnonymous)
		if err != nil {
			return
		}
		sess, err = mux.Client(ETLSConn, MuxConfig)
		if err != nil {
			err = errors.Wrapf(err, "init yamux client to %s failed", nodeID)
			return
		}
		conn, err = sess.OpenStream()
		if err != nil {
			err = errors.Wrapf(err, "open new session to %s failed", nodeID)
		}
		return
	}
	//log.WithField("poolSize", pool.Len()).Debug("session pool size")
	conn, err = pool.Get(nodeID)
	return
}

// dialToNode connects to the node with nodeID
func dialToNode(nodeID proto.NodeID) (conn net.Conn, err error) {
	return dialToNodeEx(nodeID, false)
}

// dialToNodeEx connects to the node with nodeID
func dialToNodeEx(nodeID proto.NodeID, isAnonymous bool) (conn net.Conn, err error) {
	var rawNodeID = nodeID.ToRawNodeID()
	/*
		As a common practice of PKI, we should add some randomness to the ECDHed pre-master-key
		we did that at the [ETLS](../crypto/etls) layer with a non-deterministic authenticated
		encryption input vector(random IV).

		To understand that and func "rpc.GetSharedSecretWith"
		Please refer to:
			- RFC 5297: Synthetic Initialization Vector (SIV) Authenticated Encryption
				Using the Advanced Encryption Standard (AES)
			- RFC 5246: The Transport Layer Security (TLS) Protocol Version 1.2
		Useful links:
			- https://tools.ietf.org/html/rfc5297#section-3
			- https://tools.ietf.org/html/rfc5246#section-5
			- https://www.cryptologie.net/article/340/tls-pre-master-secrets-and-master-secrets/
	*/
	symmetricKey, err := GetSharedSecretWith(rawNodeID, isAnonymous)
	if err != nil {
		return
	}

	nodeAddr, err := GetNodeAddr(rawNodeID)
	if err != nil {
		err = errors.Wrapf(err, "resolve %s failed", rawNodeID.String())
		return
	}

	cipher := etls.NewCipher(symmetricKey)
	conn, err = dial("tcp", nodeAddr, rawNodeID, cipher, isAnonymous)
	if err != nil {
		err = errors.Wrapf(err, "connect %s %s failed", rawNodeID.String(), nodeAddr)
		return
	}

	return
}

// NewClient returns a RPC client
func NewClient() *Client {
	return &Client{}
}

// initClient initializes client with connection to given addr
func initClient(addr string) (client *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return InitClientConn(conn)
}

// InitClientConn initializes client with connection to given addr
func InitClientConn(conn net.Conn) (client *Client, err error) {
	client = NewClient()
	var muxConn *mux.Stream
	muxConn, ok := conn.(*mux.Stream)
	if !ok {
		var sess *mux.Session
		sess, err = mux.Client(conn, MuxConfig)
		if err != nil {
			err = errors.Wrap(err, "init mux client failed")
			return
		}

		muxConn, err = sess.OpenStream()
		if err != nil {
			err = errors.Wrap(err, "open stream failed")
			return
		}
	}
	client.Conn = muxConn
	client.Client = rpc.NewClientWithCodec(utils.GetMsgPackClientCodec(muxConn))
	client.RemoteAddr = conn.RemoteAddr().String()

	return client, nil
}

// Close the client RPC connection
func (c *Client) Close() {
	//log.WithField("addr", c.RemoteAddr).Debug("closing client")
	_ = c.Client.Close()
}
