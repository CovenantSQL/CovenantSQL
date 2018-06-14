/*
 * Copyright 2018 The ThunderDB Authors.
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

	ec "github.com/btcsuite/btcd/btcec"
	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/etls"
	"github.com/thunderdb/ThunderDB/crypto/kms"
	"github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/route"
	"github.com/ugorji/go/codec"
)

// Client is RPC client
type Client struct {
	*rpc.Client
}

// Dial connects to a address with a Cipher
// address should be in the form of host:port
func Dial(network, address string, cipher *etls.Cipher) (c *etls.CryptoConn, err error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		log.Errorf("connect to %s failed: %s", address, err)
		return
	}
	// send NodeID + Uint256 Nonce
	nodeID, err := kms.GetLocalNodeID()
	if err != nil {
		log.Errorf("get local node id failed: %s", err)
		return
	}
	nonce, err := kms.GetLocalNonce()
	if err != nil {
		log.Errorf("get local nonce failed: %s", err)
		return
	}
	writeBuf := append(nodeID, nonce.Bytes()...)
	wrote, err := conn.Write(writeBuf)
	if err != nil || wrote != len(writeBuf) {
		log.Errorf("write node id and nonce failed: %s", err)
		return
	}

	c = etls.NewConn(conn, cipher)
	c.Conn = conn
	return
}

// DailToNode connects to the node with nodeID
func DailToNode(nodeID proto.NodeID) (conn *etls.CryptoConn, err error) {
	var nodePublicKey *ec.PublicKey
	if route.IsBPNodeID(nodeID) {
		nodePublicKey = kms.BPPublicKey
	} else {
		nodePublicKey, err = kms.GetPublicKey(nodeID)
		if err != nil {
			if err == kms.ErrKeyNotFound {
				// TODO(auxten): get node public key form  BP
			} else {
				log.Errorf("get public key failed: %s, err: %s", nodeID, err)
				return
			}
		}
	}
	localPrivateKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		log.Errorf("get local private key failed: %s", err)
		return
	}
	symmetricKey := asymmetric.GenECDHSharedSecret(localPrivateKey, nodePublicKey)

	nodeAddr, err := route.GetNodeAddr(nodeID)
	if err != nil {
		log.Errorf("resolve node id failed: %s, err: %s", nodeID, err)
		return
	}

	cipher := etls.NewCipher(symmetricKey)
	conn, err = Dial("tcp", nodeAddr, cipher)
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

// InitClient initializes client with connection to given addr
func InitClient(addr string) (client *Client, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return InitClientConn(conn)
}

// InitClientConn initializes client with connection to given addr
func InitClientConn(conn net.Conn) (client *Client, err error) {
	client = NewClient()
	client.start(conn)
	return client, nil
}

// start initializes session and set RPC codec
func (c *Client) start(conn net.Conn) {
	sess, err := yamux.Client(conn, nil)
	if err != nil {
		log.Panic(err)
	}

	clientConn, err := sess.Open()
	if err != nil {
		log.Panic(err)
		return
	}
	mh := &codec.MsgpackHandle{}
	msgpackCodec := codec.MsgpackSpecRpc.ClientCodec(clientConn, mh)
	c.Client = rpc.NewClientWithCodec(msgpackCodec)
}

// Close the client RPC connection
func (c *Client) Close() {
	c.Client.Close()
}
