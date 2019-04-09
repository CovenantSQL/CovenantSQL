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

package naconn

import (
	"bytes"
	"net"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/crypto/etls"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

const (
	// HeaderSize is the header size with ETLSHeader + NodeID + Nonce.
	HeaderSize = etls.MagicSize + hash.HashBSize + cpuminer.Uint256Size
)

// Remoter defines the interface to acquire remote node ID.
type Remoter interface {
	Remote() proto.RawNodeID
}

// NAConn defines node aware connection based on ETLS crypto connection.
type NAConn struct {
	*etls.CryptoConn
	isClient bool

	// The following fields may be rewritten during handshake.
	isAnonymous bool
	remote      proto.RawNodeID
}

// NewServerConn takes a raw connection and returns a new server side NAConn.
func NewServerConn(conn net.Conn) *NAConn {
	return &NAConn{
		CryptoConn: etls.NewConn(conn, nil), // at server side, cipher will be set during handshake
	}
}

//func NewClientConn(conn net.Conn) *NAConn {
//	return &NAConn{
//		CryptoConn:  etls.NewConn(conn, nil),
//		isClient:    true,
//		isAnonymous: true,
//	}
//}

// Remote returns the remote node ID of the NAConn.
func (c *NAConn) Remote() proto.RawNodeID {
	return c.remote
}

// Handshake does the initial handshaking according to the connection role.
func (c *NAConn) Handshake() (err error) {
	if c.isClient {
		return c.clientHandshake()
	}
	return c.serverHandshake()
}

func (c *NAConn) serverHandshake() (err error) {
	headerBuf := make([]byte, HeaderSize)
	rCount, err := c.CryptoConn.Conn.Read(headerBuf)
	if err != nil {
		err = errors.Wrap(err, "read node header error")
		return
	}

	if rCount != HeaderSize {
		err = errors.New("invalid ETLS header size")
		return
	}

	if !bytes.Equal(headerBuf[:etls.MagicSize], etls.MagicBytes[:]) {
		err = errors.New("bad ETLS header")
		return
	}

	// headerBuf len is hash.HashBSize, so there won't be any error
	idHash, _ := hash.NewHash(headerBuf[etls.MagicSize : etls.MagicSize+hash.HashBSize])
	rawNodeID := &proto.RawNodeID{Hash: *idHash}
	// TODO(auxten): compute the nonce and check difficulty
	_, _ = cpuminer.Uint256FromBytes(headerBuf[etls.MagicSize+hash.HashBSize:])

	isAnonymous := rawNodeID.IsEqual(&kms.AnonymousRawNodeID.Hash)
	symmetricKey, err := GetSharedSecretWith(defaultResolver, rawNodeID, isAnonymous)
	if err != nil {
		err = errors.Wrapf(err, "get shared secret, target: %s", rawNodeID.String())
		return
	}
	cipher := etls.NewCipher(symmetricKey)
	c.CryptoConn.Cipher = cipher // reset cipher
	c.remote = *rawNodeID
	c.isAnonymous = isAnonymous

	return
}

func (c *NAConn) clientHandshake() (err error) {
	writeBuf := make([]byte, HeaderSize)
	copy(writeBuf, etls.MagicBytes[:])
	if c.isAnonymous {
		copy(writeBuf[etls.MagicSize:], kms.AnonymousRawNodeID.AsBytes())
		copy(writeBuf[etls.MagicSize+hash.HashSize:], (&cpuminer.Uint256{}).Bytes())
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
		copy(writeBuf[etls.MagicSize:], nodeIDBytes)
		copy(writeBuf[etls.MagicSize+hash.HashSize:], nonce.Bytes())
	}
	wrote, err := c.Conn.Write(writeBuf)
	if err != nil {
		err = errors.Wrap(err, "write node id and nonce failed")
		return
	}

	if wrote != HeaderSize {
		err = errors.Errorf("write header size not match %d", wrote)
		return
	}
	return
}

// Accept takes the ownership of conn and accepts it as a NAConn.
func Accept(conn net.Conn) (*NAConn, error) {
	naconn := NewServerConn(conn)
	if err := naconn.Handshake(); err != nil {
		return nil, err
	}
	return naconn, nil
}

// Dial connects to the node with remote node id.
func Dial(remote proto.NodeID) (conn net.Conn, err error) {
	return DialEx(remote, false)
}

// DialEx connects to the node with remote node id.
func DialEx(remote proto.NodeID, isAnonymous bool) (conn net.Conn, err error) {
	var rawNodeID = remote.ToRawNodeID()
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
	symmetricKey, err := GetSharedSecretWith(defaultResolver, rawNodeID, isAnonymous)
	if err != nil {
		return
	}

	nodeAddr, err := defaultResolver.Resolve(rawNodeID)
	if err != nil {
		err = errors.Wrapf(err, "resolve %s failed", rawNodeID.String())
		return
	}

	cipher := etls.NewCipher(symmetricKey)
	iconn, err := net.Dial("tcp", nodeAddr)
	if err != nil {
		err = errors.Wrapf(err, "connect to node %s failed", nodeAddr)
		return
	}

	naconn := &NAConn{
		CryptoConn:  etls.NewConn(iconn, cipher),
		isAnonymous: isAnonymous,
		isClient:    true,
		remote:      *rawNodeID,
	}

	if err = naconn.Handshake(); err != nil {
		err = errors.Wrapf(err, "connect %s %s failed", rawNodeID.String(), nodeAddr)
		return
	}

	return naconn, nil
}
