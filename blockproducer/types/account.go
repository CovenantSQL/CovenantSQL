package types

import (
	"gitlab.com/thunderdb/ThunderDB/proto"
	"bytes"
	"gitlab.com/thunderdb/ThunderDB/utils"
	"encoding/binary"
)

// SQL Chain role type
const (
	Miner byte = iota
	Customer
)

// Account store its balance, and other mate data
type Account struct {
	Address proto.AccountAddress
	StableCoinBalance uint64
	ThunderCoinBalance uint64
	SQLChains []proto.DatabaseID
	Roles []byte
	Rating float64
}

func (a *Account) MarshalBinary() ([]byte, error) {

	buffer := bytes.NewBuffer(nil)

	err := utils.WriteElements(buffer, binary.BigEndian,
		a.Address,
		a.StableCoinBalance,
		a.ThunderCoinBalance,
		&a.SQLChains,
		&a.Roles,
		a.Rating,
	)

	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (a *Account) UnmarshalBinary(b []byte) error {
	reader := bytes.NewReader(b)
	return utils.ReadElements(reader, binary.BigEndian,
		&a.Address,
		&a.StableCoinBalance,
		&a.ThunderCoinBalance,
		&a.SQLChains,
		&a.Roles,
		&a.Rating,
	)
}

func (a *Account) AppendSQLChainAndRole(sqlChain *proto.DatabaseID, role byte) {
	a.SQLChains = append(a.SQLChains, *sqlChain)
	a.Roles = append(a.Roles, role)
}
