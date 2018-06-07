package pbtypes

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtxoEntry(t *testing.T) {
	utxoHeader := &UtxoHeader{
		Version: 1,
		PrevTxHash: &Hash{
			Hash: []byte{0x10, 0x38, 0xa1, 0x22},
		},
		Signee: &PublicKey{
			PublicKey: []byte{
				0x04, 0x11, 0xdb, 0x93, 0xe1, 0xdc, 0xdb, 0x8a,
				0x01, 0x6b, 0x49, 0x84, 0x0f, 0x8c, 0x53, 0xbc, 0x1e,
				0xb6, 0x8a, 0x38, 0x2e, 0x97, 0xb1, 0x48, 0x2e, 0xca,
				0xd7, 0xb1, 0x48, 0xa6, 0x90, 0x9a, 0x5c, 0xb2, 0xe0,
				0xea, 0xdd, 0xfb, 0x84, 0xcc, 0xf9, 0x74, 0x44, 0x64,
				0xf8, 0x2e, 0x16, 0x0b, 0xfa, 0x9b, 0x8b, 0x64, 0xf9,
				0xd4, 0xc0, 0x3f, 0x99, 0x9b, 0x86, 0x43, 0xf6, 0x56,
				0xb4, 0x12, 0xa3,
			},
		},
		Signature: &Signature{
			R: "122718002921",
			S: "192890180857",
		},
	}
	utxo := &Utxo{
		UtxoHeader: utxoHeader,
		Spent:      true,
		Amount:     1222,
	}
	utxoEntry := &UtxoEntry{
		IsCoinbase:    false,
		FromMainChain: false,
		BlockHeight:   1222,
		SparseOutputs: map[uint32]*Utxo{
			1: utxo,
		},
	}

	if utxoEntry.GetIsCoinbase() {
		t.Errorf("IsCoinbase should be false but get true")
	}
	if utxoEntry.GetFromMainChain() {
		t.Errorf("FromMainChain should be false but get true")
	}
	if utxoEntry.GetBlockHeight() != 1222 {
		t.Errorf("BlockHeight should be 1222, but get %d", utxoEntry.GetBlockHeight())
	}
	if len(utxoEntry.GetSparseOutputs()) != 1 {
		t.Errorf("Lenght of SparseOutputs should be 1, but get %d", len(utxoEntry.GetSparseOutputs()))
	}

	if !utxo.GetSpent() {
		t.Errorf("Spent should be true, but get false")
	}
	if utxo.GetAmount() != 1222 {
		t.Errorf("Amount should be 1222, but get %d", utxo.Amount)
	}

	if utxo.GetUtxoHeader().GetVersion() != 1 {
		t.Errorf("Version should be 1, but get %d", utxo.GetUtxoHeader().GetVersion())
	}

	if !bytes.Equal(utxoHeader.GetPrevTxHash().GetHash(), []byte{0x10, 0x38, 0xa1, 0x22}) {
		t.Errorf("Hash should be %v, but get %v", []byte{0x10, 0x38, 0xa1, 0x22}, utxoHeader.GetPrevTxHash())
	}
	pk := []byte{
		0x04, 0x11, 0xdb, 0x93, 0xe1, 0xdc, 0xdb, 0x8a,
		0x01, 0x6b, 0x49, 0x84, 0x0f, 0x8c, 0x53, 0xbc, 0x1e,
		0xb6, 0x8a, 0x38, 0x2e, 0x97, 0xb1, 0x48, 0x2e, 0xca,
		0xd7, 0xb1, 0x48, 0xa6, 0x90, 0x9a, 0x5c, 0xb2, 0xe0,
		0xea, 0xdd, 0xfb, 0x84, 0xcc, 0xf9, 0x74, 0x44, 0x64,
		0xf8, 0x2e, 0x16, 0x0b, 0xfa, 0x9b, 0x8b, 0x64, 0xf9,
		0xd4, 0xc0, 0x3f, 0x99, 0x9b, 0x86, 0x43, 0xf6, 0x56,
		0xb4, 0x12, 0xa3,
	}
	if !bytes.Equal(utxoHeader.GetSignee().GetPublicKey(), pk) {
		t.Errorf("PublicKey should be %v, but get %v", pk, utxoHeader.GetSignee())
	}
	R := "122718002921"
	S := "192890180857"
	if strings.Compare(R, utxoHeader.GetSignature().GetR()) != 0 {
		t.Errorf("R should be %s, but get %s", R, utxoHeader.GetSignature().GetR())
	}
	if strings.Compare(S, utxoHeader.GetSignature().GetS()) != 0 {
		t.Errorf("S should be %s, but get %s", S, utxoHeader.GetSignature().GetS())
	}

	_ = utxoEntry.String()
	utxoEntry.ProtoMessage()
	_, _ = utxoEntry.Descriptor()
	b := make([]byte, utxoEntry.XXX_Size())
	b, _ = utxoEntry.XXX_Marshal(b, true)
	_ = utxoEntry.XXX_Unmarshal(b)
	utxoEntry.XXX_DiscardUnknown()
	utxoEntry.Reset()

	_ = utxoHeader.String()
	utxoHeader.ProtoMessage()
	_, _ = utxoHeader.Descriptor()
	b = make([]byte, utxoHeader.XXX_Size())
	b, _ = utxoHeader.XXX_Marshal(b, true)
	_ = utxoHeader.XXX_Unmarshal(b)
	utxoHeader.XXX_DiscardUnknown()
	utxoHeader.Reset()

	_ = utxo.String()
	utxo.ProtoMessage()
	_, _ = utxo.Descriptor()
	b = make([]byte, utxo.XXX_Size())
	b, _ = utxo.XXX_Marshal(b, true)
	_ = utxo.XXX_Unmarshal(b)
	utxo.XXX_DiscardUnknown()
	utxo.Reset()
}

func TestTxType(t *testing.T) {
	tt := TxType_QUERY

	_ = tt.String()
	_, _ = tt.EnumDescriptor()
}

func TestSignature(t *testing.T) {
	var sig *Signature
	sig = nil
	assert.Equal(t, "", sig.GetR())
	assert.Equal(t, "", sig.GetS())

	sig = &Signature{
		R: "122718002921",
		S: "192890180857",
	}

	_ = sig.String()
	sig.ProtoMessage()
	_, _ = sig.Descriptor()
	b := make([]byte, sig.XXX_Size())
	b, _ = sig.XXX_Marshal(b, true)
	_ = sig.XXX_Unmarshal(b)
	sig.XXX_DiscardUnknown()
	sig.Reset()
}

func TestPublicKey(t *testing.T) {
	var pk *PublicKey
	pk = nil
	assert.Nil(t, pk)
	pk = &PublicKey{
		PublicKey: []byte{
			0x04, 0x11, 0xdb, 0x93, 0xe1, 0xdc, 0xdb, 0x8a,
			0x01, 0x6b, 0x49, 0x84, 0x0f, 0x8c, 0x53, 0xbc, 0x1e,
			0xb6, 0x8a, 0x38, 0x2e, 0x97, 0xb1, 0x48, 0x2e, 0xca,
			0xd7, 0xb1, 0x48, 0xa6, 0x90, 0x9a, 0x5c, 0xb2, 0xe0,
			0xea, 0xdd, 0xfb, 0x84, 0xcc, 0xf9, 0x74, 0x44, 0x64,
			0xf8, 0x2e, 0x16, 0x0b, 0xfa, 0x9b, 0x8b, 0x64, 0xf9,
			0xd4, 0xc0, 0x3f, 0x99, 0x9b, 0x86, 0x43, 0xf6, 0x56,
			0xb4, 0x12, 0xa3,
		},
	}

	_ = pk.String()
	pk.ProtoMessage()
	_, _ = pk.Descriptor()
	b := make([]byte, pk.XXX_Size())
	b, _ = pk.XXX_Marshal(b, true)
	_ = pk.XXX_Unmarshal(b)
	pk.XXX_DiscardUnknown()
	pk.Reset()
}

func TestHash(t *testing.T) {
	var h *Hash
	h = nil
	assert.Nil(t, h.GetHash())
	h = &Hash{
		Hash: []byte{0x10, 0x38, 0xa1, 0x22},
	}

	_ = h.String()
	h.ProtoMessage()
	_, _ = h.Descriptor()
	b := make([]byte, h.XXX_Size())
	b, _ = h.XXX_Marshal(b, true)
	_ = h.XXX_Unmarshal(b)
	h.XXX_DiscardUnknown()
	h.Reset()
}

func TestTx(t *testing.T) {
	var tx *Tx
	tx = nil
	assert.Nil(t, tx.GetUtxoIn())
	assert.Nil(t, tx.GetUtxoOut())
	assert.Equal(t, TxType_QUERY, tx.GetType())
	assert.Equal(t, "", tx.GetContent())

	tx = &Tx{
		UtxoIn:  []*Utxo{nil},
		UtxoOut: []*Utxo{nil},
		Type:    TxType_STORAGE,
		Content: "aaa",
	}

	assert.Equal(t, []*Utxo{nil}, tx.GetUtxoIn())
	assert.Equal(t, []*Utxo{nil}, tx.GetUtxoOut())
	assert.Equal(t, TxType_STORAGE, tx.GetType())
	assert.Equal(t, "aaa", tx.GetContent())

	_ = tx.String()
	tx.ProtoMessage()
	_, _ = tx.Descriptor()
	b := make([]byte, tx.XXX_Size())
	b, _ = tx.XXX_Marshal(b, true)
	_ = tx.XXX_Unmarshal(b)
	tx.XXX_DiscardUnknown()
	tx.Reset()
}
