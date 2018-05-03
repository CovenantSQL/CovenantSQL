package crypto

import (
	"bytes"
	"crypto/aes"
	"errors"
	"github.com/btcsuite/btcd/btcec"
)

var errInvalidPadding = errors.New("invalid PKCS#7 padding")

// Implement PKCS#7 padding with block size of 16 (AES block size).

// AddPKCSPadding adds padding to a block of data
func AddPKCSPadding(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

// RemovePKCSPadding removes padding from data that was added with addPKCSPadding
func RemovePKCSPadding(src []byte) ([]byte, error) {
	length := len(src)
	padLength := int(src[length-1])
	if padLength > aes.BlockSize || length < aes.BlockSize {
		return nil, errInvalidPadding
	}

	return src[:length-padLength], nil
}

// EncryptAndSign(inputPublicKey, inData) MAIN PROCEDURE:
//	1. newPrivateKey, newPubKey := genSecp256k1Keypair()
//	2. encKey, HMACKey := SHA512(ECDH(newPrivateKey, inputPublicKey))
//	3. PaddedIn := PKCSPadding(in)
//	4. OutBytes := IV + newPubKey + AES-256-CBC(encKey, PaddedIn) + HMAC-SHA-256(HMACKey)
func EncryptAndSign(inputPublicKey *btcec.PublicKey, inData []byte) ([]byte, error) {
	return btcec.Encrypt(inputPublicKey, inData)
}

// DecryptAndCheck(inputPrivateKey, inData) MAIN PROCEDURE:
//	1. Decrypt the inData
//  2. Verify the HMAC
func DecryptAndCheck(inputPrivateKey *btcec.PrivateKey, inData []byte) ([]byte, error) {
	return btcec.Decrypt(inputPrivateKey, inData)
}
