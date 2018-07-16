package types

import (
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"math/rand"
	"time"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"testing"
	"os"
	"io/ioutil"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
)

var (
	genesisHash = hash.Hash{}
	uuidLen = 32
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func generateRandomAccount() *Account {
	n := rand.Int31n(100) + 1
	sqlChains := generateRandomDatabaseID(n)
	roles := generateRandomBytes(n)

	h := hash.Hash{}
	rand.Read(h[:])

	account := &Account{
		Address: proto.AccountAddress(h),
		StableCoinBalance: rand.Uint64(),
		ThunderCoinBalance: rand.Uint64(),
		SQLChains: sqlChains,
		Roles: roles,
		Rating: rand.Float64(),
	}

	return account
}

func generateRandomBytes(n int32) []byte {
	s := make([]byte, n)
	for i := range s {
		s[i] = byte(rand.Int31n(2))
	}
	return s
}

func generateRandomDatabaseID(n int32) []proto.DatabaseID {
	s := make([]proto.DatabaseID, n)
	for i := range s {
		s[i] = proto.DatabaseID(randStringBytes(uuidLen))
	}
	return s
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func generateRandomBlock(parent hash.Hash, isGenesis bool) (b *Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &Block{
		SignedHeader: SignedHeader{
			Header: Header{
				Version: 0x01000000,
				Producer: proto.AccountAddress(h),
				ParentHash: parent,
				Timestamp: time.Now().UTC(),
			},
			Signee: pub,
		},
	}

	for i, n := 0, rand.Intn(10)+10; i < n; i++ {
		h := &hash.Hash{}
		rand.Read(h[:])
		b.PushTx(h)
	}

	err = b.PackAndSignBlock(priv)
	return
}

func setup() {
	rand.Seed(time.Now().UnixNano())
	rand.Read(genesisHash[:])
	f, err := ioutil.TempFile("", "keystore")

	if err != nil {
		panic(err)
	}

	f.Close()

	if err = kms.InitPublicKeyStore(f.Name(), nil); err != nil {
		panic(err)
	}

	kms.Unittest = true

	if priv, pub, err := asymmetric.GenSecp256k1KeyPair(); err == nil {
		kms.SetLocalKeyPair(priv, pub)
	} else {
		panic(err)
	}

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}
