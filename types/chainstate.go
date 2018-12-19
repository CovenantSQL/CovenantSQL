/*
 *  Copyright 2018 The CovenantSQL Authors.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/pkg/errors"
)

type ChainState struct {
	DatabaseID proto.DatabaseID
	Users      []*SQLChainUser
	Miner      []*MinerInfo
	GasPrice   uint64
	TokenType  TokenType
	MCHeight   uint32
	HeightTx   []*hash.Hash
}

func NewChainState(profile *SQLChainProfile, height uint32, createDB hash.Hash) *ChainState {
	return &ChainState{
		DatabaseID: profile.ID,
		Users:      profile.Users,
		Miner:      profile.Miners,
		GasPrice:   profile.GasPrice,
		TokenType:  profile.TokenType,
		MCHeight:   height,
		HeightTx:   []*hash.Hash{&createDB},
	}
}

func (cs *ChainState) AddAdvancePayment(transfer *Transfer, count uint32) (err error) {
	if transfer.Receiver.DatabaseID() != cs.DatabaseID {
		err = errors.Wrap(ErrNotExists, "mismatch databaseID when add advance payment")
		return
	}

	exist := false
	for _, user := range cs.Users {
		// TODO(lambda): add token type in types.Transfer
		if user.Address == transfer.Sender {
			res, flow := utils.SafeAdd(user.AdvancePayment, transfer.Amount)
			if !flow {
				exist = true
				user.AdvancePayment = res
				err = cs.updateHeightAndTx(count, transfer.Hash())
				if err != nil {
					return
				}
				break
			} else {
				err = errors.Wrap(ErrOverflow, "add advance payment overflow")
				return
			}
		}
	}

	if !exist {
		err = errors.Wrap(ErrNoSuchUser, "add advance payment for non-existing user")
	}
	return
}

func (cs *ChainState) UpdatePermission(up *UpdatePermission, count uint32) (err error) {
	if up.TargetSQLChain.DatabaseID() != cs.DatabaseID {
		err = errors.Wrap(ErrNotExists, "mismatch databaseID when update permission")
		return
	}

	newUser := true
	for _, user := range cs.Users {
		if user.Address == up.TargetUser {
			user.Permission = up.Permission
			newUser = false
			return cs.updateHeightAndTx(count, up.Hash())
		}
	}
	if newUser {
		cs.Users = append(cs.Users, &SQLChainUser{
			Address:    up.TargetUser,
			Permission: up.Permission,
		})
		return cs.updateHeightAndTx(count, up.Hash())
	}

	return
}

func (cs *ChainState) updateHeightAndTx(count uint32, tx hash.Hash) (err error) {
	if cs.MCHeight == count {
		cs.HeightTx = append(cs.HeightTx, &tx)
	} else if cs.MCHeight > count {
		err = errors.Wrap(ErrInvalidHeight, "receive old height when update height")
		return
	} else {
		cs.HeightTx = []*hash.Hash{
			&tx,
		}
		cs.MCHeight = count
	}
	return
}
