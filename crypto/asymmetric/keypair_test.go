package asymmetric

import (
	"bytes"
	"github.com/thunderdb/ThunderDB/crypto"
	"testing"
)

func TestGenSecp256k1Keypair(t *testing.T) {
	privateKey, publicKey, err := GenSecp256k1Keypair()
	if err != nil {
		t.Fatal("failed to generate private key")
	}

	in := []byte("Hey there dude. How are you doing? This is a test.")

	out, err := crypto.EncryptAndSign(publicKey, in)
	if err != nil {
		t.Fatal("failed to encrypt:", err)
	}

	dec, err := crypto.DecryptAndCheck(privateKey, out)
	if err != nil {
		t.Fatal("failed to decrypt:", err)
	}

	if !bytes.Equal(in, dec) {
		t.Error("decrypted data doesn't match original")
	}
}
