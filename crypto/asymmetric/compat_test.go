/*
 * Copyright 2026 The CovenantSQL Authors.
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

package asymmetric

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"
)

// Golden vectors recorded from the btcsuite/btcd/btcec v1 based implementation.
// Keys and signatures are persisted on chain and exchanged between nodes, and
// nodes on either side of an upgrade must derive the same ECDH shared secrets,
// so none of these may change.

const secp256k1N = "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141"

// sha256("CovenantSQL")
const compatHash = "d811217826a62e28133317dbf85e596c10c442c326ef5704b2dc99242c6b609d"

var compatKeys = []struct {
	priv, pubCompressed, pubUncompressed string
	sigR, sigS, sigDER, highSDER         string
}{
	{
		priv:            "0000000000000000000000000000000000000000000000000000000000000001",
		pubCompressed:   "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubUncompressed: "0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
		sigR:            "088c34b78b55a44154951fe9c133a88d510edeac27720919ed36ca6899a6a94f",
		sigS:            "7dc92ee2d811c69be683945b152895bf989acfdcd81d2ac455c059f1408cc137",
		sigDER:          "30440220088c34b78b55a44154951fe9c133a88d510edeac27720919ed36ca6899a6a94f02207dc92ee2d811c69be683945b152895bf989acfdcd81d2ac455c059f1408cc137",
		highSDER:        "30450220088c34b78b55a44154951fe9c133a88d510edeac27720919ed36ca6899a6a94f0221008236d11d27ee3964197c6ba4ead76a3f22140d09d72b75776a12049b8fa9800a",
	},
	{
		priv:            "fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364140",
		pubCompressed:   "0379be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
		pubUncompressed: "0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798b7c52588d95c3b9aa25b0403f1eef75702e84bb7597aabe663b82f6f04ef2777",
		sigR:            "bc30c22ea957cb9258d62a2ab3d3280415033614c20404394379d0d9581b7439",
		sigS:            "1887751f6be026ee633bfaa34b226917d97e41e3825a9432a511a3291da7e514",
		sigDER:          "3045022100bc30c22ea957cb9258d62a2ab3d3280415033614c20404394379d0d9581b743902201887751f6be026ee633bfaa34b226917d97e41e3825a9432a511a3291da7e514",
		highSDER:        "3046022100bc30c22ea957cb9258d62a2ab3d3280415033614c20404394379d0d9581b7439022100e7788ae0941fd9119cc4055cb4dd96e6e1309b032cee0c091ac0bb63b28e5c2d",
	},
	{
		priv:            "54c54aaab164f481fb3bb54d9081faba0f5981afc0fbf01da46caa330fc651e2",
		pubCompressed:   "027a55788d71a2d3d4328f3f07c3fa97d0d6341745e2c46a197629f38cb9e8c36a",
		pubUncompressed: "047a55788d71a2d3d4328f3f07c3fa97d0d6341745e2c46a197629f38cb9e8c36a80ff731e6d539295892e728adb59ef8c9fd482c01626a51a23e2767eed828f2c",
		sigR:            "fcb0c0c2458bbdbdf9c9b472dbbeeb5685ef7f515bdbeb9b9c8fbd3acf6cd914",
		sigS:            "458f3db9568ba0c2a01e40d30b1c557b320505b68518e0d5f7fd6733476a0e6c",
		sigDER:          "3045022100fcb0c0c2458bbdbdf9c9b472dbbeeb5685ef7f515bdbeb9b9c8fbd3acf6cd9140220458f3db9568ba0c2a01e40d30b1c557b320505b68518e0d5f7fd6733476a0e6c",
		highSDER:        "3046022100fcb0c0c2458bbdbdf9c9b472dbbeeb5685ef7f515bdbeb9b9c8fbd3acf6cd914022100ba70c246a9745f3d5fe1bf2cf4e3aa8388a9d7302a2fbf65c7d4f75988cc32d5",
	},
	{
		priv:            "4a1c90424ac1a06410102f467e914b0f3608a43e97c413fb6814fa462e460c3c",
		pubCompressed:   "03b02f94662ad75bf498f99ea2568902d80e05173ee881424c4eb33b86f7cd47e1",
		pubUncompressed: "04b02f94662ad75bf498f99ea2568902d80e05173ee881424c4eb33b86f7cd47e13e43c3a742c2993522f14b0cd13a0f14b8e275a1179104e18b811d443333a78b",
		sigR:            "f8b49627b0c049723fd19a58cb367895d0e788e0e843afc29dbc3ac4c2fcb7a9",
		sigS:            "5526bd43985e4ea4778c261f45aead8b43c88d88f68824eea0a0f9fac4ecba62",
		sigDER:          "3045022100f8b49627b0c049723fd19a58cb367895d0e788e0e843afc29dbc3ac4c2fcb7a902205526bd43985e4ea4778c261f45aead8b43c88d88f68824eea0a0f9fac4ecba62",
		highSDER:        "3046022100f8b49627b0c049723fd19a58cb367895d0e788e0e843afc29dbc3ac4c2fcb7a9022100aad942bc67a1b15b8873d9e0ba51527376e64f5db8c07b4d1f3164920b4986df",
	},
}

var compatECDH = []struct {
	name, priv, peerPub, secret string
}{
	{
		name:    "full length",
		priv:    "54c54aaab164f481fb3bb54d9081faba0f5981afc0fbf01da46caa330fc651e2",
		peerPub: "03b02f94662ad75bf498f99ea2568902d80e05173ee881424c4eb33b86f7cd47e1",
		secret:  "849f3cc9c9cc6779f8c65f56385d4987e621aacdf5f77a48ac99cc8e70b677d8",
	},
	{
		// The shared X coordinate has a leading zero byte, which is stripped (31 bytes).
		name:    "leading zero stripped",
		priv:    "54c54aaab164f481fb3bb54d9081faba0f5981afc0fbf01da46caa330fc651e2",
		peerPub: "0348cc15d121d2f02878b1b59cd0d23a85e43133ff2425905736defbacb5aa4e57",
		secret:  "c527c4eab73b60b4171404934f5c3e7e330967b0abe833ab742b9754cca062",
	},
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

func mustBig(t *testing.T, s string) *big.Int {
	t.Helper()
	i, ok := new(big.Int).SetString(s, 16)
	if !ok {
		t.Fatalf("bad big int %q", s)
	}
	return i
}

func TestCompatKeys(t *testing.T) {
	for _, v := range compatKeys {
		priv, pub := PrivKeyFromBytes(mustHex(t, v.priv))
		if got := hex.EncodeToString(priv.Serialize()); got != v.priv {
			t.Errorf("private key serialize: got %s, want %s", got, v.priv)
		}
		if got := hex.EncodeToString(pub.Serialize()); got != v.pubCompressed {
			t.Errorf("public key serialize: got %s, want %s", got, v.pubCompressed)
		}
		if !pub.IsEqual(priv.PubKey()) {
			t.Errorf("PubKey() of %s differs from PrivKeyFromBytes public key", v.priv)
		}

		uncompressed := mustHex(t, v.pubUncompressed)
		wantX := new(big.Int).SetBytes(uncompressed[1:33])
		wantY := new(big.Int).SetBytes(uncompressed[33:])
		for _, in := range []string{v.pubCompressed, v.pubUncompressed} {
			parsed, err := ParsePubKey(mustHex(t, in))
			if err != nil {
				t.Fatalf("parse public key %s: %v", in, err)
			}
			if parsed.X.Cmp(wantX) != 0 || parsed.Y.Cmp(wantY) != 0 {
				t.Errorf("parse public key %s: got (%x, %x), want (%x, %x)", in, parsed.X, parsed.Y, wantX, wantY)
			}
			if !parsed.IsEqual(pub) {
				t.Errorf("parsed public key %s is not equal to derived key", in)
			}
		}
	}
}

func TestCompatSignatures(t *testing.T) {
	hash := mustHex(t, compatHash)
	n := mustBig(t, secp256k1N)
	for _, v := range compatKeys {
		priv, pub := PrivKeyFromBytes(mustHex(t, v.priv))
		r, s := mustBig(t, v.sigR), mustBig(t, v.sigS)

		sig, err := priv.Sign(hash)
		if err != nil {
			t.Fatalf("sign with %s: %v", v.priv, err)
		}
		if sig.R.Cmp(r) != 0 || sig.S.Cmp(s) != 0 {
			t.Errorf("sign with %s: got (%x, %x), want (%x, %x)", v.priv, sig.R, sig.S, r, s)
		}
		if got := hex.EncodeToString(sig.Serialize()); got != v.sigDER {
			t.Errorf("signature serialize: got %s, want %s", got, v.sigDER)
		}
		if !sig.Verify(hash, pub) {
			t.Errorf("signature by %s does not verify", v.priv)
		}

		parsed, err := ParseSignature(mustHex(t, v.sigDER))
		if err != nil {
			t.Fatalf("parse signature %s: %v", v.sigDER, err)
		}
		if !parsed.IsEqual(sig) {
			t.Errorf("parse signature %s: got (%x, %x), want (%x, %x)", v.sigDER, parsed.R, parsed.S, r, s)
		}

		// A high S value is serialized in its canonical low S form.
		highS := new(big.Int).Sub(n, s)
		high := &Signature{R: r, S: highS}
		if got := hex.EncodeToString(high.Serialize()); got != v.sigDER {
			t.Errorf("high S signature serialize: got %s, want %s", got, v.sigDER)
		}

		// Parsing keeps a high S value as is.
		parsedHigh, err := ParseSignature(mustHex(t, v.highSDER))
		if err != nil {
			t.Fatalf("parse high S signature %s: %v", v.highSDER, err)
		}
		if parsedHigh.R.Cmp(r) != 0 || parsedHigh.S.Cmp(highS) != 0 {
			t.Errorf("parse high S signature %s: got (%x, %x), want (%x, %x)", v.highSDER, parsedHigh.R, parsedHigh.S, r, highS)
		}

		// Bytes after the DER encoded length are ignored.
		for _, extra := range []int{1, 4, 30} {
			in := append(mustHex(t, v.sigDER), make([]byte, extra)...)
			parsedTrailing, err := ParseSignature(in)
			if err != nil {
				t.Fatalf("parse signature %s with %d trailing bytes: %v", v.sigDER, extra, err)
			}
			if !parsedTrailing.IsEqual(sig) {
				t.Errorf("parse signature %s with %d trailing bytes: got (%x, %x), want (%x, %x)",
					v.sigDER, extra, parsedTrailing.R, parsedTrailing.S, r, s)
			}
		}
	}
}

func TestCompatECDH(t *testing.T) {
	for _, v := range compatECDH {
		priv, _ := PrivKeyFromBytes(mustHex(t, v.priv))
		peer, err := ParsePubKey(mustHex(t, v.peerPub))
		if err != nil {
			t.Fatalf("%s: parse peer public key: %v", v.name, err)
		}
		if got := GenECDHSharedSecret(priv, peer); !bytes.Equal(got, mustHex(t, v.secret)) {
			t.Errorf("%s: got %x, want %s", v.name, got, v.secret)
		}
	}
}
