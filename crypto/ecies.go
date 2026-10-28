// Copyright (c) 2015-2016 The btcsuite developers
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

// Adapted from btcec/ciphering.go of github.com/btcsuite/btcd v0.0.0-20190614013741-962a206e94e9,
// which btcec/v2 no longer provides. The ciphertext format must not change, otherwise data
// encrypted by earlier releases can no longer be decrypted.

package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"io"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
)

var (
	// errInvalidMAC occurs when Message Authentication Check (MAC) fails
	// during decryption. This happens because of either invalid private key or
	// corrupt ciphertext.
	errInvalidMAC = errors.New("invalid mac hash")

	// errInputTooShort occurs when the input ciphertext to the Decrypt
	// function is less than 134 bytes long.
	errInputTooShort = errors.New("ciphertext too short")

	// errUnsupportedCurve occurs when the first two bytes of the encrypted
	// text aren't 0x02CA (= 712 = secp256k1, from OpenSSL).
	errUnsupportedCurve = errors.New("unsupported curve")

	errInvalidXLength = errors.New("invalid X length, must be 32")
	errInvalidYLength = errors.New("invalid Y length, must be 32")

	// 0x02CA = 714
	ciphCurveBytes = [2]byte{0x02, 0xCA}
	// 0x20 = 32
	ciphCoordLength = [2]byte{0x00, 0x20}
)

// EncryptAndSign (inputPublicKey, inData) MAIN PROCEDURE:
//  1. newPrivateKey, newPubKey := genSecp256k1Keypair()
//  2. encKey, HMACKey := SHA512(ECDH(newPrivateKey, inputPublicKey))
//  3. PaddedIn := PKCSPadding(in)
//  4. OutBytes := IV + newPubKey + AES-256-CBC(encKey, PaddedIn) + HMAC-SHA-256(HMACKey)
func EncryptAndSign(inputPublicKey *asymmetric.PublicKey, inData []byte) ([]byte, error) {
	ephemeral, ephemeralPub, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		return nil, err
	}
	ecdhKey := asymmetric.GenECDHSharedSecret(ephemeral, inputPublicKey)
	derivedKey := sha512.Sum512(ecdhKey)
	keyE := derivedKey[:32]
	keyM := derivedKey[32:]

	paddedIn := AddPKCSPadding(inData)
	// IV + Curve params/X/Y + padded plaintext/ciphertext + HMAC-256
	out := make([]byte, aes.BlockSize+70+len(paddedIn)+sha256.Size)
	iv := out[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	// start writing public key
	offset := aes.BlockSize

	// curve and X length
	copy(out[offset:offset+4], append(ciphCurveBytes[:], ciphCoordLength[:]...))
	offset += 4
	// X
	ephemeralPub.X.FillBytes(out[offset : offset+32])
	offset += 32
	// Y length
	copy(out[offset:offset+2], ciphCoordLength[:])
	offset += 2
	// Y
	ephemeralPub.Y.FillBytes(out[offset : offset+32])
	offset += 32

	// start encryption
	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(out[offset:len(out)-sha256.Size], paddedIn)

	// start HMAC-SHA-256
	hm := hmac.New(sha256.New, keyM)
	hm.Write(out[:len(out)-sha256.Size])          // everything is hashed
	copy(out[len(out)-sha256.Size:], hm.Sum(nil)) // write checksum

	return out, nil
}

// DecryptAndCheck (inputPrivateKey, inData) MAIN PROCEDURE:
//  1. Decrypt the inData
//  2. Verify the HMAC.
func DecryptAndCheck(inputPrivateKey *asymmetric.PrivateKey, inData []byte) ([]byte, error) {
	// IV + Curve params/X/Y + 1 block + HMAC-256
	if len(inData) < aes.BlockSize+70+aes.BlockSize+sha256.Size {
		return nil, errInputTooShort
	}

	// read iv
	iv := inData[:aes.BlockSize]
	offset := aes.BlockSize

	// start reading pubkey
	if !bytes.Equal(inData[offset:offset+2], ciphCurveBytes[:]) {
		return nil, errUnsupportedCurve
	}
	offset += 2

	if !bytes.Equal(inData[offset:offset+2], ciphCoordLength[:]) {
		return nil, errInvalidXLength
	}
	offset += 2

	xBytes := inData[offset : offset+32]
	offset += 32

	if !bytes.Equal(inData[offset:offset+2], ciphCoordLength[:]) {
		return nil, errInvalidYLength
	}
	offset += 2

	yBytes := inData[offset : offset+32]
	offset += 32

	pb := make([]byte, 65)
	pb[0] = byte(0x04) // uncompressed
	copy(pb[1:33], xBytes)
	copy(pb[33:], yBytes)
	// check if (X, Y) lies on the curve and create a Pubkey if it does
	pubkey, err := asymmetric.ParsePubKey(pb)
	if err != nil {
		return nil, err
	}

	// check for cipher text length
	if (len(inData)-aes.BlockSize-offset-sha256.Size)%aes.BlockSize != 0 {
		return nil, errInvalidPadding // not padded to 16 bytes
	}

	// read hmac
	messageMAC := inData[len(inData)-sha256.Size:]

	// generate shared secret
	ecdhKey := asymmetric.GenECDHSharedSecret(inputPrivateKey, pubkey)
	derivedKey := sha512.Sum512(ecdhKey)
	keyE := derivedKey[:32]
	keyM := derivedKey[32:]

	// verify mac
	hm := hmac.New(sha256.New, keyM)
	hm.Write(inData[:len(inData)-sha256.Size]) // everything is hashed
	expectedMAC := hm.Sum(nil)
	if !hmac.Equal(messageMAC, expectedMAC) {
		return nil, errInvalidMAC
	}

	// start decryption
	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	// same length as ciphertext
	plaintext := make([]byte, len(inData)-offset-sha256.Size)
	mode.CryptBlocks(plaintext, inData[offset:len(inData)-sha256.Size])

	return RemovePKCSPadding(plaintext)
}
