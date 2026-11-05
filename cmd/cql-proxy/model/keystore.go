/*
 * Copyright 2019 The CovenantSQL Authors.
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

package model

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp/v3"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/utils"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
)

// DeveloperPrivateKey defines the private key object of developer.
type DeveloperPrivateKey struct {
	ID        int64                  `db:"id"`
	Developer int64                  `db:"developer_id"`
	Account   utils.AccountAddress   `db:"account"`
	RawKey    []byte                 `db:"key_string"`
	Created   int64                  `db:"created"`
	Key       *asymmetric.PrivateKey `db:"-"`
}

// LoadPrivateKey loads the private key from raw key bytes.
func (p *DeveloperPrivateKey) LoadPrivateKey() (err error) {
	p.Key, err = kms.DecodePrivateKey(p.RawKey, []byte{})
	if err != nil {
		err = fmt.Errorf("load private key failed: %w", err)
	}
	return
}

// GetPrivateKey load private key of account from proxy database.
func GetPrivateKey(db *gorp.DbMap, developer int64, account utils.AccountAddress) (p *DeveloperPrivateKey, err error) {
	if p, err = GetAccount(db, developer, account); err != nil || p == nil {
		if err != nil {
			err = fmt.Errorf("get private key failed: %w", err)
		}
		return
	}

	// validate key password
	if err = p.LoadPrivateKey(); err != nil {
		err = fmt.Errorf("decode private key failed: %w", err)
		p = nil
	}

	return
}

// AddNewPrivateKey generates new private key for developer.
func AddNewPrivateKey(db *gorp.DbMap, developer int64) (p *DeveloperPrivateKey, err error) {
	dbMap := db
	privateKey, pubKey, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		err = fmt.Errorf("generate secp256k1 keypair failed: %w", err)
		return
	}

	keyBytes, err := kms.EncodePrivateKey(privateKey, []byte{})
	if err != nil {
		err = fmt.Errorf("encode private key failed: %w", err)
		return
	}

	account, err := crypto.PubKeyHash(pubKey)
	if err != nil {
		err = fmt.Errorf("convert public key to wallet address failed: %w", err)
		return
	}

	p = &DeveloperPrivateKey{
		Developer: developer,
		Account:   utils.NewAccountAddress(account),
		RawKey:    keyBytes,
		Created:   time.Now().Unix(),
		Key:       privateKey,
	}

	err = dbMap.Insert(p)
	if err != nil {
		err = fmt.Errorf("add new private key failed: %w", err)
	}

	return
}

// SavePrivateKey saves existing private key to developer keys object.
func SavePrivateKey(db *gorp.DbMap, developer int64, key *asymmetric.PrivateKey) (
	p *DeveloperPrivateKey, err error) {
	dbMap := db
	exists := true

	account, err := crypto.PubKeyHash(key.PubKey())
	if err != nil {
		err = fmt.Errorf("convert public key to wallet address failed: %w", err)
		return
	}

	accountAddr := utils.NewAccountAddress(account)

	if p, err = GetAccount(db, developer, accountAddr); err != nil {
		// not exists
		err = nil
		p = &DeveloperPrivateKey{
			Developer: developer,
			Account:   accountAddr,
			Key:       key,
		}

		exists = false
	}

	if p.RawKey, err = kms.EncodePrivateKey(key, []byte{}); err != nil {
		err = fmt.Errorf("encode private key failed: %w", err)
		p = nil
		return
	}

	p.Created = time.Now().Unix()

	if exists {
		_, err = dbMap.Update(p)
		if err != nil {
			err = fmt.Errorf("update private key info failed: %w", err)
		}
	} else {
		err = dbMap.Insert(p)
		if err != nil {
			err = fmt.Errorf("add private key info failed: %w", err)
		}
	}

	return
}

// DeletePrivateKey removes the keypair from proxy database.
func DeletePrivateKey(db *gorp.DbMap, developer int64, account utils.AccountAddress) (
	p *DeveloperPrivateKey, err error) {
	p, err = GetPrivateKey(db, developer, account)
	if err != nil {
		err = fmt.Errorf("get private key failed: %w", err)
		return
	}

	_, err = db.Delete(p)
	if err != nil {
		err = fmt.Errorf("delete private key failed: %w", err)
	}

	return
}

// GetAccount returns the account object of specified keypair.
func GetAccount(db *gorp.DbMap, developer int64, account utils.AccountAddress) (p *DeveloperPrivateKey, err error) {
	err = db.SelectOne(&p,
		`SELECT * FROM "private_keys" WHERE "account" = ? AND "developer_id" = ? LIMIT 1`,
		account, developer)
	if err != nil {
		err = fmt.Errorf("get account info failed: %w", err)
	}
	return
}

// GetAllAccounts returns all accounts of developer.
func GetAllAccounts(db *gorp.DbMap, developer int64) (keys []*DeveloperPrivateKey, err error) {
	_, err = db.Select(&keys,
		`SELECT * FROM "private_keys" WHERE "developer_id" = ?`, developer)
	if err != nil {
		err = fmt.Errorf("get all developer keypairs failed: %w", err)
	}
	return
}

// GetAccountByID return account object by keypair id.
func GetAccountByID(db *gorp.DbMap, developer int64, id int64) (p *DeveloperPrivateKey, err error) {
	err = db.SelectOne(&p,
		`SELECT * FROM "private_keys" WHERE "id" = ? AND "developer_id" = ? LIMIT 1`,
		id, developer)
	if err != nil {
		err = fmt.Errorf("get account info failed: %w", err)
	}
	return
}
