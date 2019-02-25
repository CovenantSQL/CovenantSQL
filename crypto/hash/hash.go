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

package hash

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"

	hsp "github.com/CovenantSQL/HashStablePack/marshalhash"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// HashSize of array used to store hashes.  See Hash.
const HashSize = 32

// MaxHashStringSize is the maximum length of a Hash hash string.
const MaxHashStringSize = HashSize * 2

// ErrHashStrSize describes an error that indicates the caller specified a hash
// string that has too many characters.
var ErrHashStrSize = fmt.Errorf("max hash string length is %v bytes", MaxHashStringSize)

// Hash is used in several of the bitcoin messages and common structures.  It
// typically represents the double sha256 of data.
type Hash [HashSize]byte

// String returns the Hash as the hexadecimal string of the byte-reversed
// hash.
func (h Hash) String() string {
	for i := 0; i < HashSize/2; i++ {
		h[i], h[HashSize-1-i] = h[HashSize-1-i], h[i]
	}
	return hex.EncodeToString(h[:])
}

// Short returns the hexadecimal string of the first `n` reversed byte(s).
func (h Hash) Short(n int) string {
	for i := 0; i < HashSize/2; i++ {
		h[i], h[HashSize-1-i] = h[HashSize-1-i], h[i]
	}
	var l = HashSize
	if n < l {
		l = n
	}
	return hex.EncodeToString(h[:l])
}

// AsBytes returns internal bytes of hash.
func (h Hash) AsBytes() []byte {
	return h[:]
}

// CloneBytes returns a copy of the bytes which represent the hash as a byte
// slice.
//
// NOTE: It is generally cheaper to just slice the hash directly thereby reusing
// the same bytes rather than calling this method.
func (h *Hash) CloneBytes() []byte {
	newHash := make([]byte, HashSize)
	copy(newHash, h[:])

	return newHash
}

// MarshalHash marshals for hash.
func (h *Hash) MarshalHash() (o []byte, err error) {
	return h.CloneBytes(), nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message.
func (h *Hash) Msgsize() (s int) {
	return hsp.BytesPrefixSize + HashSize
}

// SetBytes sets the bytes which represent the hash.  An error is returned if
// the number of bytes passed in is not HashSize.
func (h *Hash) SetBytes(newHash []byte) error {
	nhlen := len(newHash)
	if nhlen != HashSize {
		return fmt.Errorf("invalid hash length of %v, want %v", nhlen,
			HashSize)
	}
	copy(h[:], newHash)

	return nil
}

// IsEqual returns true if target is the same as hash.
func (h *Hash) IsEqual(target *Hash) bool {
	if h == nil && target == nil {
		return true
	}
	if h == nil || target == nil {
		return false
	}
	return *h == *target
}

// Difficulty returns the leading Zero **bit** count of Hash in binary.
//  return -1 indicate the Hash pointer is nil.
func (h *Hash) Difficulty() (difficulty int) {
	if h == nil {
		return -1
	}

	for i := range *h {
		v := (*h)[HashSize-i-1]
		if v != byte(0) {
			difficulty = 8 * i
			difficulty += bits.LeadingZeros8(v)
			return
		}
	}
	return HashSize * 8
}

// MarshalJSON implements the json.Marshaler interface.
func (h Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (h *Hash) UnmarshalJSON(data []byte) (err error) {
	var s string
	if err = json.Unmarshal(data, &s); err != nil {
		return
	}
	if err = Decode(h, s); err != nil {
		return
	}
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (h Hash) MarshalYAML() (interface{}, error) {
	return h.String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (h *Hash) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	// load hash
	err := Decode(h, str)
	if err != nil {
		log.WithError(err).Error("unmarshal YAML failed")
		return err
	}
	return nil
}

// NewHash returns a new Hash from a byte slice.  An error is returned if
// the number of bytes passed in is not HashSize.
func NewHash(newHash []byte) (*Hash, error) {
	var sh Hash
	err := sh.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &sh, err
}

// NewHashFromStr creates a Hash from a hash string.  The string should be
// the hexadecimal string of a byte-reversed hash, but any missing characters
// result in zero padding at the end of the Hash.
func NewHashFromStr(hash string) (*Hash, error) {
	ret := new(Hash)
	err := Decode(ret, hash)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// Decode decodes the byte-reversed hexadecimal string encoding of a Hash to a
// destination.
func Decode(dst *Hash, src string) error {
	// Return error if hash string is too long.
	if len(src) > MaxHashStringSize {
		return ErrHashStrSize
	}

	// Hex decoder expects the hash to be a multiple of two.  When not, pad
	// with a leading zero.
	var srcBytes []byte
	if len(src)%2 == 0 {
		srcBytes = []byte(src)
	} else {
		srcBytes = make([]byte, 1+len(src))
		srcBytes[0] = '0'
		copy(srcBytes[1:], src)
	}

	// Hex decode the source bytes to a temporary destination.
	var reversedHash Hash
	_, err := hex.Decode(reversedHash[HashSize-hex.DecodedLen(len(srcBytes)):], srcBytes)
	if err != nil {
		return err
	}

	// Reverse copy from the temporary hash to destination.  Because the
	// temporary was zeroed, the written result will be correctly padded.
	for i, b := range reversedHash[:HashSize/2] {
		dst[i], dst[HashSize-1-i] = reversedHash[HashSize-1-i], b
	}

	return nil
}
