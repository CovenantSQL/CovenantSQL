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

// Package etls implements "Enhanced Transport Layer Security", but more efficient
// than TLS used in https.
// example can be found in test case.
package etls

import (
	"bytes"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	// MagicSize is the ETLS magic header size.
	MagicSize = 2
)

var (
	// MagicBytes is the ETLS connection magic header.
	MagicBytes = [MagicSize]byte{0xC0, 0x4E}
)

// CryptoConn implements net.Conn and Cipher interface.
type CryptoConn struct {
	net.Conn
	*Cipher
}

// NewConn returns a new CryptoConn.
func NewConn(c net.Conn, cipher *Cipher) *CryptoConn {
	return &CryptoConn{
		Conn:   c,
		Cipher: cipher,
	}
}

// Dial connects to a address with a Cipher
// address should be in the form of host:port.
func Dial(network, address string, cipher *Cipher) (c *CryptoConn, err error) {
	conn, err := net.DialTimeout(network, address, conf.TCPDialTimeout)
	if err != nil {
		log.WithField("addr", address).WithError(err).Error("connect failed")
		return
	}
	c = NewConn(conn, cipher)
	return
}

// Read iv and Encrypted data.
func (c *CryptoConn) Read(b []byte) (n int, err error) {
	if c.decStream == nil {
		buf := make([]byte, c.info.ivLen+MagicSize)
		if _, err = io.ReadFull(c.Conn, buf); err != nil {
			log.WithError(err).Info("read full failed")
			return
		}
		iv := buf[:c.info.ivLen]
		header := buf[c.info.ivLen:]
		if err = c.initDecrypt(iv); err != nil {
			return
		}
		c.decrypt(header, header)
		if !bytes.Equal(header[:MagicSize], MagicBytes[:]) {
			err = errors.New("bad stream ETLS header")
			return
		}
	}

	cipherData := make([]byte, len(b))

	n, err = c.Conn.Read(cipherData)
	if err != nil {
		return
	}
	if n > 0 {
		c.decrypt(b[0:n], cipherData[0:n])
	}
	return
}

// Write iv and Encrypted data.
func (c *CryptoConn) Write(b []byte) (n int, err error) {
	var iv []byte
	if c.encStream == nil {
		iv, err = c.initEncrypt()
		if err != nil {
			return
		}
	}

	if iv != nil {
		ivHeader := make([]byte, len(iv)+MagicSize)
		// Put initialization vector in buffer, do a single write to send both
		// iv and data.
		copy(ivHeader, iv)
		c.encrypt(ivHeader[len(iv):], MagicBytes[:])
		_, err = c.Conn.Write(ivHeader)
		if err != nil {
			return
		}
	}

	cipherData := make([]byte, len(b))
	c.encrypt(cipherData, b)
	n, err = c.Conn.Write(cipherData)
	return
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *CryptoConn) Close() error {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

// LocalAddr returns the local network address.
func (c *CryptoConn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *CryptoConn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection.
// A zero value for t means Read and Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *CryptoConn) SetDeadline(t time.Time) error {
	return c.Conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying connection.
// A zero value for t means Read will not time out.
func (c *CryptoConn) SetReadDeadline(t time.Time) error {
	return c.Conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying connection.
// A zero value for t means Write will not time out.
// After a Write has timed out, the TLS state is corrupt and all future writes
// will return the same error.
func (c *CryptoConn) SetWriteDeadline(t time.Time) error {
	return c.Conn.SetWriteDeadline(t)
}
