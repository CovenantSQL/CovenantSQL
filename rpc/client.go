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

	"io/ioutil"

	"github.com/CovenantSQL/CovenantSQL/crypto/etls"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/hashicorp/yamux"
	"github.com/ugorji/go/codec"
)

// Client is RPC client
type Client struct {
	*rpc.Client
	RemoteAddr string
	Conn       net.Conn
}

var (
	// YamuxConfig holds the default Yamux config
	YamuxConfig *yamux.Config
	// DefaultDialer holds the default dialer of SessionPool
	DefaultDialer func(nodeID proto.NodeID) (conn net.Conn, err error)
)

func init() {
	YamuxConfig = yamux.DefaultConfig()
	YamuxConfig.LogOutput = ioutil.Discard
	DefaultDialer = dialToNode
}

// dial connects to a address with a Cipher
// address should be in the form of host:port
func dial(network, address string, remoteNodeID *proto.RawNodeID, cipher *etls.Cipher, isAnonymous bool) (c *etls.CryptoConn, err error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		log.Errorf("connect to %s failed: %s", address, err)
		return
	}
	var writeBuf []byte
	if isAnonymous {
		writeBuf = append(kms.AnonymousRawNodeID.CloneBytes(), (&cpuminer.Uint256{}).Bytes()...)
	} else {
		// send NodeID + Uint256 Nonce
		var nodeIDBytes []byte
		var nonce *cpuminer.Uint256
		nodeIDBytes, err = kms.GetLocalNodeIDBytes()
		if err != nil {
			log.Errorf("get local node id failed: %s", err)
			return
		}
		nonce, err = kms.GetLocalNonce()
		if err != nil {
			log.Errorf("get local nonce failed: %s", err)
			return
		}
		writeBuf = append(nodeIDBytes, nonce.Bytes()...)
	}
	wrote, err := conn.Write(writeBuf)
	if err != nil || wrote != len(writeBuf) {
		log.Errorf("write node id and nonce failed: %s", err)
		return
	}

	c = etls.NewConn(conn, cipher, remoteNodeID)
	return
}

// DialToNode ties use connection in pool, if fails then connects to the node with nodeID
func DialToNode(nodeID proto.NodeID, pool *SessionPool, isAnonymous bool) (conn net.Conn, err error) {
	if pool == nil || isAnonymous {
		var ETLSConn net.Conn
		var sess *yamux.Session
		ETLSConn, err = dialToNodeEx(nodeID, isAnonymous)
		if err != nil {
			log.Errorf("dialToNode failed: %s", err)
			return
		}
		sess, err = yamux.Client(ETLSConn, YamuxConfig)
		if err != nil {
			log.Errorf("init yamux client failed: %s", err)
			return
		}
		conn, err = sess.Open()
		if err != nil {
			log.Errorf("open new session failed: %s", err)
		}
		return
	}
	log.Debugf("session pool len: %d", pool.Len())
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
		log.Errorf("get shared secret for %s failed: %s", rawNodeID.ToNodeID(), err)
		return
	}

	nodeAddr, err := GetNodeAddr(rawNodeID)
	if err != nil {
		log.Errorf("resolve node %x failed, err: %s", *rawNodeID, err)
		return
	}

	cipher := etls.NewCipher(symmetricKey)
	conn, err = dial("tcp", nodeAddr, rawNodeID, cipher, isAnonymous)
	if err != nil {
		log.Errorf("connect to %s: %s", nodeAddr, err)
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
	var muxConn *yamux.Stream
	muxConn, ok := conn.(*yamux.Stream)
	if !ok {
		var sess *yamux.Session
		sess, err = yamux.Client(conn, YamuxConfig)
		if err != nil {
			log.Errorf("init yamux client failed: %v", err)
			return
		}

		muxConn, err = sess.OpenStream()
		if err != nil {
			log.Errorf("open stream failed: %v", err)
			return
		}
	}
	client.Conn = muxConn
	mh := &codec.MsgpackHandle{
		WriteExt:    true,
		RawToString: true,
	}
	msgpackCodec := codec.MsgpackSpecRpc.ClientCodec(muxConn, mh)
	client.Client = rpc.NewClientWithCodec(msgpackCodec)
	client.RemoteAddr = conn.RemoteAddr().String()

	return client, nil
}

// Close the client RPC connection
func (c *Client) Close() {
	log.Debugf("closing %s", c.RemoteAddr)
	c.Client.Close()
}
