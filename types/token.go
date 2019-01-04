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

//go:generate hsp

// TokenType defines token's type
type TokenType int32

const (
	// Particle defines covenantsql's token
	Particle TokenType = iota
	// Wave defines covenantsql's token
	Wave
	// Ether defines Ethereum.
	Ether
	// EOS defines EOS.
	EOS
	// Bitcoin defines Bitcoin.
	Bitcoin
	// SupportTokenNumber defines the number of token covenantsql supports
	SupportTokenNumber
)

// TokenList lists supporting token.
var TokenList = map[TokenType]string{
	Particle: "Particle",
	Wave:     "Wave",
	Ether:    "Ether",
	EOS:      "EOS",
	Bitcoin:  "Bitcoin",
}

// String returns token's symbol.
func (t TokenType) String() string {
	if t < 0 || t >= SupportTokenNumber {
		return "Unknown"
	}

	return TokenList[t]
}

// FromString returns token's number.
func FromString(t string) TokenType {
	var i TokenType
	for ; i < SupportTokenNumber; i++ {
		if TokenList[i] == t {
			return i
		}
	}
	return -1
}

// Listed returns if the token is listed in list.
func (t *TokenType) Listed() bool {
	return (*t) >= 0 && *t < SupportTokenNumber
}
