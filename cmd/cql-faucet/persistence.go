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

package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/utils/log"

	// Load sqlite3 database driver.
	_ "github.com/CovenantSQL/go-sqlite3-encrypt"
)

// Persistence defines the persistence api for faucet service.
type Persistence struct {
	db                *sql.DB
	accountDailyQuota uint
	addressDailyQuota uint
	tokenAmount       int64
}

// applicationRecord defines single record for verification.
type applicationRecord struct {
	id          string
	rowID       int64
	account     string
	email       string
	tokenAmount int64 // covenantsql could not store uint64 value, use int64 instead
	createTime  time.Time
}

func (r *applicationRecord) asMap() (result map[string]interface{}) {
	result = make(map[string]interface{})

	result["id"] = r.id
	result["rowID"] = r.rowID
	result["account"] = r.account
	result["email"] = r.email
	result["tokenAmount"] = r.tokenAmount
	result["createTime"] = r.createTime.String()

	return
}

// NewPersistence returns a new applyToken persistence api.
func NewPersistence(faucetCfg *Config) (p *Persistence, err error) {
	p = &Persistence{
		accountDailyQuota: faucetCfg.AccountDailyQuota,
		addressDailyQuota: faucetCfg.AddressDailyQuota,
		tokenAmount:       faucetCfg.FaucetAmount,
	}

	// connect database
	if faucetCfg.LocalDatabase {
		// treat DatabaseID as sqlite3 file
		dbPath := filepath.Join(conf.GConf.WorkingRoot, faucetCfg.DatabaseID)
		if p.db, err = sql.Open("sqlite3", dbPath); err != nil {
			return
		}
	} else {
		cfg := client.NewConfig()
		cfg.DatabaseID = faucetCfg.DatabaseID

		if p.db, err = sql.Open("covenantsql", cfg.FormatDSN()); err != nil {
			return
		}
	}

	// init database
	err = p.initDB()

	return
}

func (p *Persistence) initDB() (err error) {
	_, err = p.db.ExecContext(context.Background(),
		`CREATE TABLE IF NOT EXISTS faucet_records (
				id text unique,
				account text, 
				email text,
				amount bigint, 
				ctime datetime
			  )`)
	return
}

func (p *Persistence) checkAccountLimit(account string) (err error) {
	timeOfDayStart := time.Now().UTC().Format("2006-01-02 00:00:00")

	// account limit check
	row := p.db.QueryRowContext(context.Background(),
		`SELECT COUNT(1) AS cnt FROM faucet_records
		WHERE ctime >= ? AND account = ?`,
		timeOfDayStart, account)

	var result uint

	err = row.Scan(&result)
	if err != nil {
		return
	}

	if result >= p.accountDailyQuota {
		// quota exceeded
		log.WithField("account", account).Error("daily account quota exceeded")
		return ErrAccountQuotaExceeded
	}

	return
}

func (p *Persistence) checkEmailLimit(email string) (err error) {
	timeOfDayStart := time.Now().UTC().Format("2006-01-02 00:00:00")

	// account limit check
	row := p.db.QueryRowContext(context.Background(),
		`SELECT COUNT(1) AS cnt FROM faucet_records
		WHERE ctime >= ? AND email = ?`,
		timeOfDayStart, email)

	var result uint

	err = row.Scan(&result)
	if err != nil {
		return
	}

	if result >= p.addressDailyQuota {
		// quota exceeded
		log.WithField("email", email).Error("daily email quota exceeded")
		return ErrEmailQuotaExceeded
	}

	return
}

// addRecord record a new token applyToken to CovenantSQL database.
func (p *Persistence) addRecord(account string, email string) (applicationID string, err error) {
	// generate uuid
	applicationID = uuid.Must(uuid.NewV4()).String()
	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	// enqueue
	_, err = p.db.ExecContext(context.Background(),
		`INSERT INTO faucet_records (
				id,
				account,
				email,
				amount,
				ctime
			  ) VALUES (?, ?, ?, ?, ?)`,
		applicationID, account, email, p.tokenAmount, now)

	if err != nil {
		log.WithFields(log.Fields{
			"account": account,
			"email":   email,
		}).Errorf("enqueue applyToken failed: %v", err)

		err = ErrEnqueueApplication
	}

	return
}
