package types

import (
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"math/rand"
	"time"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

var (
	genesisHash = hash.Hash{}
	uuidLen = 32
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func GenerateRandomAccount() *Account {
	rand.Seed(time.Now().UnixNano())
	rand.Read(genesisHash[:])
	n := rand.Int31n(100) + 1
	sqlChains := GenerateRandomDatabaseID(n)
	roles := GenerateRandomBytes(n)

	account := &Account{
		Address: proto.AccountAddress(genesisHash),
		StableCoinBalance: rand.Uint64(),
		ThunderCoinBalance: rand.Uint64(),
		SQLChains: sqlChains,
		Roles: roles,
		Rating: rand.Float64(),
	}

	return account
}

func GenerateRandomBytes(n int32) []byte {
	rand.Seed(time.Now().UnixNano())
	s := make([]byte, n)
	for i := range s {
		s[i] = byte(rand.Int31n(2))
	}
	return s
}

func GenerateRandomDatabaseID(n int32) []proto.DatabaseID {
	rand.Seed(time.Now().UnixNano())
	s := make([]proto.DatabaseID, n)
	for i := range s {
		s[i] = proto.DatabaseID(RandStringBytes(uuidLen))
	}
	return s
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
