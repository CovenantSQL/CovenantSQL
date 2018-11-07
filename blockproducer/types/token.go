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

package types

import (
	"bytes"
	"encoding/binary"

	"github.com/CovenantSQL/HashStablePack/marshalhash"
)

// SupportTokenNumber defines the number of token covenantsql supports
const SupportTokenNumber int32 = 4

// Token defines token's number.
var Token = [SupportTokenNumber]string{
	"Particle",
	"Ether",
	"EOS",
	"Bitcoin",
}

// TokenType defines token's type
type TokenType int32

const (
	// Particle defines covenantsql's token
	Particle TokenType = iota
	// Ether defines Ethereum.
	Ether
	// EOS defines EOS.
	EOS
	// Bitcoin defines Bitcoin.
	Bitcoin
)

// String returns token's symbol.
func (t TokenType) String() string {
	if t < 0 || int32(t) >= SupportTokenNumber {
		return "Unknown"
	}

	return Token[int(t)]
}

// FromString returns token's number.
func FromString(t string) TokenType {
	for i := range Token {
		if t == Token[i] {
			return TokenType(i)
		}
	}
	return -1
}

// Listed returns if the token is listed in list.
func (t *TokenType) Listed() bool {
	return (*t) >= 0 && int32(*t) < SupportTokenNumber
}

// MarshalHash marshals for hash.
func (t *TokenType) MarshalHash() (o []byte, err error) {
	var binBuf bytes.Buffer
	binary.Write(&binBuf, binary.BigEndian, t)
	return binBuf.Bytes(), nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message.
func (t *TokenType) Msgsize() (s int) {
	return marshalhash.BytesPrefixSize + 4
}
