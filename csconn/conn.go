package csconn

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
	// HeaderSize is the header size with ETLSHeader + NodeID + Nonce
	HeaderSize = etls.MagicSize + hash.HashBSize + cpuminer.Uint256Size
)

type CSConn struct {
	*etls.CryptoConn
	isClient bool

	// The following fields may be rewritten during handshake.
	isAnonymous  bool
	remoteNodeID proto.RawNodeID
}

func NewServerConn(conn net.Conn) *CSConn {
	return &CSConn{
		CryptoConn: etls.NewConn(conn, nil), // at server side, cipher will be set during handshake
	}
}

//func NewClientConn(conn net.Conn) *CSConn {
//	return &CSConn{
//		CryptoConn:  etls.NewConn(conn, nil),
//		isClient:    true,
//		isAnonymous: true,
//	}
//}

func (c *CSConn) RemoteNodeID() proto.RawNodeID {
	return c.remoteNodeID
}

func (c *CSConn) Handshake() (err error) {
	if c.isClient {
		return c.clientHandshake()
	}
	return c.serverHandshake()
}

func (c *CSConn) serverHandshake() (err error) {
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
	cpuminer.Uint256FromBytes(headerBuf[etls.MagicSize+hash.HashBSize:])

	isAnonymous := rawNodeID.IsEqual(&kms.AnonymousRawNodeID.Hash)
	symmetricKey, err := GetSharedSecretWith(defaultResolver, rawNodeID, isAnonymous)
	if err != nil {
		err = errors.Wrapf(err, "get shared secret, target: %s", rawNodeID.String())
		return
	}
	cipher := etls.NewCipher(symmetricKey)
	c.CryptoConn.Cipher = cipher
	c.remoteNodeID = *rawNodeID
	c.isAnonymous = isAnonymous

	return
}

func (c *CSConn) clientHandshake() (err error) {
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

// Accept takes the ownership of conn and accepts it as a CSConn.
func Accept(conn net.Conn) (*CSConn, error) {
	csconn := NewServerConn(conn)
	if err := csconn.Handshake(); err != nil {
		return nil, err
	}
	return csconn, nil
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

	csconn := &CSConn{
		CryptoConn:   etls.NewConn(iconn, cipher),
		isAnonymous:  isAnonymous,
		isClient:     true,
		remoteNodeID: *rawNodeID,
	}

	if err = csconn.Handshake(); err != nil {
		err = errors.Wrapf(err, "connect %s %s failed", rawNodeID.String(), nodeAddr)
		return
	}

	return csconn, nil
}
