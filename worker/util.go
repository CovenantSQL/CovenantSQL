/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"gitlab.com/thunderdb/ThunderDB/crypto/signature"
	"gitlab.com/thunderdb/ThunderDB/types"
	"github.com/btcsuite/btcd/btcec"
	"math/big"
	"gitlab.com/thunderdb/ThunderDB/sqlchain"
	"time"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

func publicKeyToPB(publicKey *signature.PublicKey) *types.PublicKey {
	return &types.PublicKey{
		PublicKey: publicKey.Serialize(),
	}
}

func publicKeyFromPB(pb *types.PublicKey) (*signature.PublicKey, error) {
	return signature.ParsePubKey(pb.GetPublicKey(), btcec.S256())
}

func signatureToPB(sig *signature.Signature) *types.Signature {
	if sig == nil {
		return nil
	}

	return &types.Signature{
		S: sig.S.String(),
		R: sig.R.String(),
	}
}

func signatureFromPB(pb *types.Signature) (*signature.Signature, error) {
	pr := new(big.Int)
	ps := new(big.Int)

	pr, ok := pr.SetString(pb.GetR(), 10)

	if !ok {
		return nil, sqlchain.ErrFieldConversion
	}

	ps, ok = ps.SetString(pb.GetS(), 10)

	if !ok {
		return nil, sqlchain.ErrFieldConversion
	}

	return &signature.Signature{
		R: pr,
		S: ps,
	}, nil
}

func timeToTimestamp(t time.Time) int64 {
	return t.UnixNano()
}

func timeFromTimestamp(tm int64) time.Time {
	return time.Unix(0, tm).UTC()
}

func hashToPB(h *hash.Hash) *types.Hash {
	return &types.Hash{Hash: h[:]}
}

func hashFromPB(pb *types.Hash, h *hash.Hash) (err error) {
	if len(pb.GetHash()) != hash.HashSize {
		return ErrFieldLength
	}

	copy(h[:], pb.GetHash())
	return
}

type hasMarshal interface {
	marshal() ([]byte, error)
}

func verifyHash(data hasMarshal, h *hash.Hash) error {
	var newHash hash.Hash
	buildHash(data, &newHash)
	if !newHash.IsEqual(h) {
		return ErrHashVerification
	}
	return nil
}

func buildHash(data hasMarshal, h *hash.Hash) (err error) {
	var buffer []byte
	if buffer, err = data.marshal(); err != nil {
		return
	}

	newHash := hash.DoubleHashH(buffer)
	copy(h[:], newHash[:])
	return
}
