/*
 * Copyright 2018 The CovenantSQL Authors.
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

package blockproducer

import (
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/pkg/errors"
	"github.com/ulule/deepcopier"
)

var (
	sqlchainPeriod   uint64 = 60 * 24 * 30
	sqlchainGasPrice uint64 = 10
)

// TODO(leventeliu): lock optimization.

type metaState struct {
	dirty, readonly *metaIndex
}

func newMetaState() *metaState {
	return &metaState{
		dirty:    newMetaIndex(),
		readonly: newMetaIndex(),
	}
}

func (s *metaState) loadAccountObject(k proto.AccountAddress) (o *accountObject, loaded bool) {
	if o, loaded = s.dirty.accounts[k]; loaded {
		if o == nil {
			loaded = false
		}
		return
	}
	if o, loaded = s.readonly.accounts[k]; loaded {
		return
	}
	return
}

func (s *metaState) loadOrStoreAccountObject(
	k proto.AccountAddress, v *accountObject) (o *accountObject, loaded bool,
) {
	if o, loaded = s.dirty.accounts[k]; loaded && o != nil {
		return
	}
	if o, loaded = s.readonly.accounts[k]; loaded {
		return
	}
	s.dirty.accounts[k] = v
	return
}

func (s *metaState) loadAccountStableBalance(addr proto.AccountAddress) (b uint64, loaded bool) {
	var o *accountObject
	defer func() {
		log.WithFields(log.Fields{
			"account": addr.String(),
			"balance": b,
			"loaded":  loaded,
		}).Debug("queried stable account")
	}()

	if o, loaded = s.dirty.accounts[addr]; loaded && o != nil {
		b = o.TokenBalance[types.Particle]
		return
	}
	if o, loaded = s.readonly.accounts[addr]; loaded {
		b = o.TokenBalance[types.Particle]
		return
	}
	return
}

func (s *metaState) loadAccountCovenantBalance(addr proto.AccountAddress) (b uint64, loaded bool) {
	var o *accountObject
	defer func() {
		log.WithFields(log.Fields{
			"account": addr.String(),
			"balance": b,
			"loaded":  loaded,
		}).Debug("queried covenant account")
	}()

	if o, loaded = s.dirty.accounts[addr]; loaded && o != nil {
		b = o.TokenBalance[types.Wave]
		return
	}
	if o, loaded = s.readonly.accounts[addr]; loaded {
		b = o.TokenBalance[types.Wave]
		return
	}
	return
}

func (s *metaState) storeBaseAccount(k proto.AccountAddress, v *accountObject) (err error) {
	log.WithFields(log.Fields{
		"addr":    k.String(),
		"account": v,
	}).Debug("store account")
	// Since a transfer tx may create an empty receiver account, this method should try to cover
	// the side effect.
	if ao, ok := s.loadOrStoreAccountObject(k, v); ok {
		if ao.Account.NextNonce != 0 {
			err = ErrAccountExists
			return
		}
		var (
			cb = ao.TokenBalance[types.Wave]
			sb = ao.TokenBalance[types.Particle]
		)
		if err = safeAdd(&cb, &v.Account.TokenBalance[types.Wave]); err != nil {
			return
		}
		if err = safeAdd(&sb, &v.Account.TokenBalance[types.Particle]); err != nil {
			return
		}
		ao.TokenBalance[types.Wave] = cb
		ao.TokenBalance[types.Particle] = sb
	}
	return
}

func (s *metaState) loadSQLChainObject(k proto.DatabaseID) (o *sqlchainObject, loaded bool) {
	if o, loaded = s.dirty.databases[k]; loaded {
		if o == nil {
			loaded = false
		}
		return
	}
	if o, loaded = s.readonly.databases[k]; loaded {
		return
	}
	return
}

func (s *metaState) loadOrStoreSQLChainObject(
	k proto.DatabaseID, v *sqlchainObject) (o *sqlchainObject, loaded bool,
) {
	if o, loaded = s.dirty.databases[k]; loaded && o != nil {
		return
	}
	if o, loaded = s.readonly.databases[k]; loaded {
		return
	}
	s.dirty.databases[k] = v
	return
}

func (s *metaState) loadProviderObject(k proto.AccountAddress) (o *providerObject, loaded bool) {
	if o, loaded = s.dirty.provider[k]; loaded {
		if o == nil {
			loaded = false
		}
		return
	}
	if o, loaded = s.readonly.provider[k]; loaded {
		return
	}
	return
}

func (s *metaState) loadOrStoreProviderObject(k proto.AccountAddress, v *providerObject) (o *providerObject, loaded bool) {
	if o, loaded = s.dirty.provider[k]; loaded && o != nil {
		return
	}
	if o, loaded = s.readonly.provider[k]; loaded {
		return
	}
	s.dirty.provider[k] = v
	return
}

func (s *metaState) deleteAccountObject(k proto.AccountAddress) {
	// Use a nil pointer to mark a deletion, which will be later used by commit procedure.
	s.dirty.accounts[k] = nil
}

func (s *metaState) deleteSQLChainObject(k proto.DatabaseID) {
	// Use a nil pointer to mark a deletion, which will be later used by commit procedure.
	s.dirty.databases[k] = nil
}

func (s *metaState) deleteProviderObject(k proto.AccountAddress) {
	// Use a nil pointer to mark a deletion, which will be later used by commit procedure.
	s.dirty.provider[k] = nil
}

func (s *metaState) commit() {
	for k, v := range s.dirty.accounts {
		if v != nil {
			// New/update object
			s.readonly.accounts[k] = v
		} else {
			// Delete object
			delete(s.readonly.accounts, k)
		}
	}
	for k, v := range s.dirty.databases {
		if v != nil {
			// New/update object
			s.readonly.databases[k] = v
		} else {
			// Delete object
			delete(s.readonly.databases, k)
		}
	}
	for k, v := range s.dirty.provider {
		if v != nil {
			// New/update object
			s.readonly.provider[k] = v
		} else {
			// Delete object
			delete(s.readonly.provider, k)
		}
	}
	// Clean dirty map
	s.dirty = newMetaIndex()
	return
}

func (s *metaState) clean() {
	s.dirty = newMetaIndex()
}

func (s *metaState) increaseAccountToken(k proto.AccountAddress, amount uint64, tokenType types.TokenType) error {
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			err := errors.Wrap(ErrAccountNotFound, "increase stable balance fail")
			return err
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeAdd(&dst.Account.TokenBalance[tokenType], &amount)
}

func (s *metaState) decreaseAccountToken(k proto.AccountAddress, amount uint64, tokenType types.TokenType) error {
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			err := errors.Wrap(ErrAccountNotFound, "increase stable balance fail")
			return err
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeSub(&dst.Account.TokenBalance[tokenType], &amount)
}

func (s *metaState) increaseAccountStableBalance(k proto.AccountAddress, amount uint64) error {
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			err := errors.Wrap(ErrAccountNotFound, "increase stable balance fail")
			return err
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeAdd(&dst.Account.TokenBalance[types.Particle], &amount)
}

func (s *metaState) decreaseAccountStableBalance(k proto.AccountAddress, amount uint64) error {
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			return ErrAccountNotFound
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeSub(&dst.Account.TokenBalance[types.Particle], &amount)
}

func (s *metaState) transferAccountStableBalance(
	sender, receiver proto.AccountAddress, amount uint64) (err error,
) {
	if sender == receiver || amount == 0 {
		return
	}

	// Create empty receiver account if not found
	s.loadOrStoreAccountObject(receiver, &accountObject{Account: types.Account{Address: receiver}})

	var (
		so, ro     *accountObject
		sd, rd, ok bool
	)

	// Load sender and receiver objects
	if so, sd = s.dirty.accounts[sender]; !sd {
		if so, ok = s.readonly.accounts[sender]; !ok {
			err = ErrAccountNotFound
			return
		}
	}
	if ro, rd = s.dirty.accounts[receiver]; !rd {
		if ro, ok = s.readonly.accounts[receiver]; !ok {
			err = ErrAccountNotFound
			return
		}
	}

	// Try transfer
	var (
		sb = so.TokenBalance[types.Particle]
		rb = ro.TokenBalance[types.Particle]
	)
	if err = safeSub(&sb, &amount); err != nil {
		return
	}
	if err = safeAdd(&rb, &amount); err != nil {
		return
	}

	// Proceed transfer
	if !sd {
		var cpy = &accountObject{}
		deepcopier.Copy(&so.Account).To(&cpy.Account)
		so = cpy
		s.dirty.accounts[sender] = cpy
	}
	if !rd {
		var cpy = &accountObject{}
		deepcopier.Copy(&ro.Account).To(&cpy.Account)
		ro = cpy
		s.dirty.accounts[receiver] = cpy
	}
	so.TokenBalance[types.Particle] = sb
	ro.TokenBalance[types.Particle] = rb
	return
}

func (s *metaState) increaseAccountCovenantBalance(k proto.AccountAddress, amount uint64) error {
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			err := errors.Wrap(ErrAccountNotFound, "increase covenant balance fail")
			return err
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeAdd(&dst.Account.TokenBalance[types.Wave], &amount)
}

func (s *metaState) decreaseAccountCovenantBalance(k proto.AccountAddress, amount uint64) error {
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			return ErrAccountNotFound
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeSub(&dst.Account.TokenBalance[types.Wave], &amount)
}

func (s *metaState) createSQLChain(addr proto.AccountAddress, id proto.DatabaseID) error {
	if _, ok := s.dirty.accounts[addr]; !ok {
		if _, ok := s.readonly.accounts[addr]; !ok {
			return ErrAccountNotFound
		}
	}
	if _, ok := s.dirty.databases[id]; ok {
		return ErrDatabaseExists
	} else if _, ok := s.readonly.databases[id]; ok {
		return ErrDatabaseExists
	}
	s.dirty.databases[id] = &sqlchainObject{
		SQLChainProfile: types.SQLChainProfile{
			ID:     id,
			Owner:  addr,
			Miners: make([]*types.MinerInfo, 0),
			Users: []*types.SQLChainUser{
				{
					Address:    addr,
					Permission: types.Admin,
				},
			},
		},
	}
	return nil
}

func (s *metaState) addSQLChainUser(
	k proto.DatabaseID, addr proto.AccountAddress, perm types.UserPermission) (_ error,
) {
	var (
		src, dst *sqlchainObject
		ok       bool
	)
	if dst, ok = s.dirty.databases[k]; !ok {
		if src, ok = s.readonly.databases[k]; !ok {
			return ErrDatabaseNotFound
		}
		dst = &sqlchainObject{}
		deepcopier.Copy(&src.SQLChainProfile).To(&dst.SQLChainProfile)
		s.dirty.databases[k] = dst
	}
	for _, v := range dst.Users {
		if v.Address == addr {
			return ErrDatabaseUserExists
		}
	}
	dst.SQLChainProfile.Users = append(dst.SQLChainProfile.Users, &types.SQLChainUser{
		Address:    addr,
		Permission: perm,
	})
	return
}

func (s *metaState) deleteSQLChainUser(k proto.DatabaseID, addr proto.AccountAddress) error {
	var (
		src, dst *sqlchainObject
		ok       bool
	)
	if dst, ok = s.dirty.databases[k]; !ok {
		if src, ok = s.readonly.databases[k]; !ok {
			return ErrDatabaseNotFound
		}
		dst = &sqlchainObject{}
		deepcopier.Copy(&src.SQLChainProfile).To(&dst.SQLChainProfile)
		s.dirty.databases[k] = dst
	}
	for i, v := range dst.Users {
		if v.Address == addr {
			last := len(dst.Users) - 1
			dst.Users[i] = dst.Users[last]
			dst.Users[last] = nil
			dst.Users = dst.Users[:last]
		}
	}
	return nil
}

func (s *metaState) alterSQLChainUser(
	k proto.DatabaseID, addr proto.AccountAddress, perm types.UserPermission) (_ error,
) {
	var (
		src, dst *sqlchainObject
		ok       bool
	)
	if dst, ok = s.dirty.databases[k]; !ok {
		if src, ok = s.readonly.databases[k]; !ok {
			return ErrDatabaseNotFound
		}
		dst = &sqlchainObject{}
		deepcopier.Copy(&src.SQLChainProfile).To(&dst.SQLChainProfile)
		s.dirty.databases[k] = dst
	}
	for _, v := range dst.Users {
		if v.Address == addr {
			v.Permission = perm
		}
	}
	return
}

func (s *metaState) nextNonce(addr proto.AccountAddress) (nonce pi.AccountNonce, err error) {
	var (
		o      *accountObject
		loaded bool
	)
	if o, loaded = s.dirty.accounts[addr]; !loaded {
		if o, loaded = s.readonly.accounts[addr]; !loaded {
			err = ErrAccountNotFound
			log.WithFields(log.Fields{
				"addr": addr.String(),
			}).WithError(err).Error("unexpected error")
			return
		}
	}
	nonce = o.Account.NextNonce
	return
}

func (s *metaState) increaseNonce(addr proto.AccountAddress) (err error) {
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[addr]; !ok {
		if src, ok = s.readonly.accounts[addr]; !ok {
			return ErrAccountNotFound
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[addr] = dst
	}
	dst.NextNonce++
	return
}

func (s *metaState) applyBilling(tx *types.Billing) (err error) {
	for i, v := range tx.Receivers {
		// Create empty receiver account if not found
		s.loadOrStoreAccountObject(*v, &accountObject{Account: types.Account{Address: *v}})

		if err = s.increaseAccountCovenantBalance(*v, tx.Fees[i]); err != nil {
			return
		}
		if err = s.increaseAccountStableBalance(*v, tx.Rewards[i]); err != nil {
			return
		}
	}
	return
}

func (s *metaState) updateProviderList(tx *types.ProvideService) (err error) {
	sender, err := crypto.PubKeyHash(tx.Signee)
	if err != nil {
		err = errors.Wrap(err, "updateProviderList failed")
		return
	}

	// deposit
	var (
		minDeposit = conf.GConf.MinProviderDeposit
	)
	if err = s.decreaseAccountStableBalance(sender, minDeposit); err != nil {
		return
	}
	pp := types.ProviderProfile{
		Provider:      sender,
		Space:         tx.Space,
		Memory:        tx.Memory,
		LoadAvgPerCPU: tx.LoadAvgPerCPU,
		TargetUser:    tx.TargetUser,
		Deposit:       minDeposit,
		GasPrice:      tx.GasPrice,
		NodeID:        tx.NodeID,
	}
	s.loadOrStoreProviderObject(sender, &providerObject{ProviderProfile: pp})
	return
}

func (s *metaState) matchProvidersWithUser(tx *types.CreateDatabase) (err error) {
	sender, err := crypto.PubKeyHash(tx.Signee)
	if err != nil {
		err = errors.Wrap(err, "matchProviders failed")
		return
	}
	if sender != tx.Owner {
		err = errors.Wrapf(ErrInvalidSender, "match failed with real sender: %s, sender: %s",
			sender.String(), tx.Owner.String())
		return
	}

	if tx.GasPrice <= 0 {
		err = ErrInvalidGasPrice
		return
	}

	var (
		minAdvancePayment = uint64(tx.GasPrice) * uint64(conf.GConf.QPS) *
			uint64(conf.GConf.Period) * uint64(len(tx.ResourceMeta.TargetMiners))
	)
	if tx.AdvancePayment < minAdvancePayment {
		err = ErrInsufficientAdvancePayment
		return
	}

	var (
		miners = make([]*types.MinerInfo, len(tx.ResourceMeta.TargetMiners))
	)

	for i := range tx.ResourceMeta.TargetMiners {
		if po, loaded := s.loadProviderObject(tx.ResourceMeta.TargetMiners[i]); !loaded {
			log.WithFields(log.Fields{
				"miner_addr": tx.ResourceMeta.TargetMiners[i].String(),
				"user_addr":  sender.String(),
			}).Error(err)
			err = ErrNoSuchMiner
			break
		} else {
			if po.TargetUser != sender {
				log.WithFields(log.Fields{
					"miner_addr": tx.ResourceMeta.TargetMiners[i].String(),
					"user_addr":  sender.String(),
				}).Error(ErrMinerUserNotMatch)
				err = ErrMinerUserNotMatch
				break
			}
			if po.GasPrice > tx.GasPrice {
				err = ErrGasPriceMismatch
				break
			}
			miners[i] = &types.MinerInfo{
				Address: po.Provider,
				NodeID:  po.NodeID,
				Deposit: po.Deposit,
			}
		}
	}
	if err != nil {
		return
	}

	// generate new sqlchain id and address
	dbID := proto.FromAccountAndNonce(tx.Owner, uint32(tx.Nonce))
	dbAddr, err := dbID.AccountAddress()
	if err != nil {
		err = errors.Wrapf(err, "unexpected error when convert dbid: %v", dbID)
		return
	}
	// generate userinfo
	var all = minAdvancePayment
	err = safeAdd(&all, &tx.AdvancePayment)
	if err != nil {
		return
	}
	err = s.decreaseAccountToken(sender, all, tx.TokenType)
	if err != nil {
		return
	}
	users := make([]*types.SQLChainUser, 1)
	users[0] = &types.SQLChainUser{
		Address:        sender,
		Permission:     types.Admin,
		Deposit:        minAdvancePayment,
		AdvancePayment: tx.AdvancePayment,
	}
	// generate genesis block
	gb, err := s.generateGenesisBlock(*dbID, tx.ResourceMeta)
	if err != nil {
		log.WithFields(log.Fields{
			"dbID":         dbID,
			"resourceMeta": tx.ResourceMeta,
		}).WithError(err).Error("unexpected error")
		return err
	}

	// create sqlchain
	sp := &types.SQLChainProfile{
		ID:        *dbID,
		Address:   dbAddr,
		Period:    sqlchainPeriod,
		GasPrice:  sqlchainGasPrice,
		TokenType: types.Particle,
		Owner:     sender,
		Users:     users,
		Genesis:   gb,
		Miners:    miners[:],
	}

	if _, loaded := s.loadSQLChainObject(*dbID); loaded {
		err = errors.Wrapf(ErrDatabaseExists, "database exists: %s", string(*dbID))
		return
	}
	s.loadOrStoreAccountObject(dbAddr, &accountObject{
		Account: types.Account{Address: dbAddr},
	})
	s.loadOrStoreSQLChainObject(*dbID, &sqlchainObject{SQLChainProfile: *sp})
	for _, miner := range tx.ResourceMeta.TargetMiners {
		s.deleteProviderObject(miner)
	}
	return
}

func (s *metaState) updatePermission(tx *types.UpdatePermission) (err error) {
	sender, err := crypto.PubKeyHash(tx.Signee)
	if err != nil {
		log.WithFields(log.Fields{
			"tx": tx.Hash().String(),
		}).WithError(err).Error("unexpected err")
		return
	}
	if sender == tx.TargetUser {
		err = errors.Wrap(ErrInvalidSender, "user cannot update its permission by itself")
		return
	}
	so, loaded := s.loadSQLChainObject(tx.TargetSQLChain.DatabaseID())
	if !loaded {
		log.WithFields(log.Fields{
			"dbID": tx.TargetSQLChain.DatabaseID(),
		}).WithError(ErrDatabaseNotFound).Error("unexpected error in updatePermission")
		return ErrDatabaseNotFound
	}
	if tx.Permission >= types.NumberOfUserPermission {
		log.WithFields(log.Fields{
			"permission": tx.Permission,
			"dbID":       tx.TargetSQLChain.DatabaseID(),
		}).WithError(ErrInvalidPermission).Error("unexpected error in updatePermission")
		return ErrInvalidPermission
	}

	// check whether sender is admin and find targetUser
	isAdmin := false
	targetUserIndex := -1
	for i, u := range so.Users {
		isAdmin = isAdmin || (sender == u.Address && u.Permission == types.Admin)
		if tx.TargetUser == u.Address {
			targetUserIndex = i
		}
	}

	if !isAdmin {
		log.WithFields(log.Fields{
			"sender": sender,
			"dbID":   tx.TargetSQLChain,
		}).WithError(ErrAccountPermissionDeny).Error("unexpected error in updatePermission")
		return ErrAccountPermissionDeny
	}

	// update targetUser's permission
	if targetUserIndex == -1 {
		u := types.SQLChainUser{
			Address:    tx.TargetUser,
			Permission: tx.Permission,
			Status:     types.Normal,
		}
		so.Users = append(so.Users, &u)
	} else {
		so.Users[targetUserIndex].Permission = tx.Permission
	}
	return
}

func (s *metaState) updateKeys(tx *types.IssueKeys) (err error) {
	sender := tx.GetAccountAddress()
	so, loaded := s.loadSQLChainObject(tx.TargetSQLChain.DatabaseID())
	if !loaded {
		log.WithFields(log.Fields{
			"dbID": tx.TargetSQLChain.DatabaseID(),
		}).WithError(ErrDatabaseNotFound).Error("unexpected error in updateKeys")
		return ErrDatabaseNotFound
	}

	// check sender's permission
	isAdmin := false
	for _, user := range so.Users {
		if sender == user.Address && user.Permission == types.Admin {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		log.WithFields(log.Fields{
			"sender": sender,
			"dbID":   tx.TargetSQLChain,
		}).WithError(ErrAccountPermissionDeny).Error("unexpected error in updateKeys")
		return ErrAccountPermissionDeny
	}

	// update miner's key
	keyMap := make(map[proto.AccountAddress]string)
	for i := range tx.MinerKeys {
		keyMap[tx.MinerKeys[i].Miner] = tx.MinerKeys[i].EncryptionKey
	}
	for _, miner := range so.Miners {
		if key, ok := keyMap[miner.Address]; ok {
			miner.EncryptionKey = key
		}
	}
	return
}

func (s *metaState) updateBilling(tx *types.UpdateBilling) (err error) {
	sqlchainObj, loaded := s.loadSQLChainObject(tx.Receiver.DatabaseID())
	if !loaded {
		err = errors.Wrap(ErrDatabaseNotFound, "update billing failed")
		return
	}

	if sqlchainObj.GasPrice == 0 {
		return
	}

	// pending income to income
	for _, miner := range sqlchainObj.Miners {
		miner.ReceivedIncome += miner.PendingIncome
	}

	var (
		costMap = make(map[proto.AccountAddress]uint64)
		userMap = make(map[proto.AccountAddress]map[proto.AccountAddress]uint64)
	)
	for _, userCost := range tx.Users {
		costMap[userCost.User] = userCost.Cost
		if _, ok := userMap[userCost.User]; !ok {
			userMap[userCost.User] = make(map[proto.AccountAddress]uint64)
		}
		for _, minerIncome := range userCost.Miners {
			userMap[userCost.User][minerIncome.Miner] += minerIncome.Income
		}
	}
	for _, user := range sqlchainObj.Users {
		if user.AdvancePayment >= costMap[user.Address]*sqlchainObj.GasPrice {
			user.AdvancePayment -= costMap[user.Address] * sqlchainObj.GasPrice
			for _, miner := range sqlchainObj.Miners {
				miner.PendingIncome += userMap[user.Address][miner.Address] * sqlchainObj.GasPrice
			}
		} else {
			rate := 1 - float64(user.AdvancePayment)/float64(costMap[user.Address]*sqlchainObj.GasPrice)
			user.AdvancePayment = 0
			user.Status = types.Arrears
			for _, miner := range sqlchainObj.Miners {
				income := userMap[user.Address][miner.Address] * sqlchainObj.GasPrice
				minerIncome := uint64(float64(income) * rate)
				miner.PendingIncome += minerIncome
				for i := range miner.UserArrears {
					miner.UserArrears[i].Arrears += (income - minerIncome)
				}
			}
		}
	}
	return
}

func (s *metaState) applyTransaction(tx pi.Transaction) (err error) {
	switch t := tx.(type) {
	case *types.Transfer:
		realSender, err := crypto.PubKeyHash(t.Signee)
		if err != nil {
			err = errors.Wrap(err, "applyTx failed")
			return err
		}
		if realSender != t.Sender {
			err = errors.Wrapf(ErrInvalidSender,
				"applyTx failed: real sender %s, sender %s", realSender.String(), t.Sender.String())
			// TODO(lambda): update test cases and return err
			log.Debug(err)
		}
		err = s.transferAccountStableBalance(t.Sender, t.Receiver, t.Amount)
	case *types.Billing:
		err = s.applyBilling(t)
	case *types.BaseAccount:
		err = s.storeBaseAccount(t.Address, &accountObject{Account: t.Account})
	case *types.ProvideService:
		err = s.updateProviderList(t)
	case *types.CreateDatabase:
		err = s.matchProvidersWithUser(t)
	case *types.UpdatePermission:
		err = s.updatePermission(t)
	case *types.IssueKeys:
		err = s.updateKeys(t)
	case *types.UpdateBilling:
		err = s.updateBilling(t)
	case *pi.TransactionWrapper:
		// call again using unwrapped transaction
		err = s.applyTransaction(t.Unwrap())
	default:
		err = ErrUnknownTransactionType
	}
	return
}

func (s *metaState) generateGenesisBlock(dbID proto.DatabaseID, resourceMeta types.ResourceMeta) (genesisBlock *types.Block, err error) {
	// TODO(xq262144): following is stub code, real logic should be implemented in the future
	emptyHash := hash.Hash{}

	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	genesisBlock = &types.Block{
		SignedHeader: types.SignedHeader{
			Header: types.Header{
				Version:     0x01000000,
				Producer:    nodeID,
				GenesisHash: emptyHash,
				ParentHash:  emptyHash,
				Timestamp:   time.Now().UTC(),
			},
		},
	}

	err = genesisBlock.PackAndSignBlock(privKey)

	return
}

func (s *metaState) apply(t pi.Transaction) (err error) {
	// NOTE(leventeliu): bypass pool in this method.
	var (
		addr  = t.GetAccountAddress()
		nonce = t.GetAccountNonce()
	)
	// Check account nonce
	var nextNonce pi.AccountNonce
	if nextNonce, err = s.nextNonce(addr); err != nil {
		if t.GetTransactionType() != pi.TransactionTypeBaseAccount {
			return
		}
		// Consider the first nonce 0
		err = nil
	}
	if nextNonce != nonce {
		err = ErrInvalidAccountNonce
		log.WithFields(log.Fields{
			"actual":   nonce,
			"expected": nextNonce,
		}).WithError(err).Debug("nonce not match during transaction apply")
		return
	}
	// Try to apply transaction to metaState
	if err = s.applyTransaction(t); err != nil {
		log.WithError(err).Debug("apply transaction failed")
		return
	}
	if err = s.increaseNonce(addr); err != nil {
		return
	}
	return
}

func (s *metaState) makeCopy() *metaState {
	return &metaState{
		dirty:    newMetaIndex(),
		readonly: s.readonly.deepCopy(),
	}
}
