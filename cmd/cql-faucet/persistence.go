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

// State defines the token application request state.
type State int

const (
	// StateApplication represents application request initial state.
	StateApplication State = iota
	// StateVerified represents the application request has already been verified.
	StateVerified
	// StateDispensed represents the application request has been fulfilled and tokens are dispensed.
	StateDispensed
	// StateFailed represents the application is invalid or maybe quota exceeded.
	StateFailed
	// StateUnknown represents invalid state
	StateUnknown
)

func (s State) String() string {
	switch s {
	case StateApplication:
		return "StateApplication"
	case StateVerified:
		return "StateVerified"
	case StateDispensed:
		return "StateDispensed"
	case StateFailed:
		return "StateFailed"
	case StateUnknown:
		return "StateUnknown"
	}

	return ""
}

// Persistence defines the persistence api for faucet service.
type Persistence struct {
	db                *sql.DB
	accountDailyQuota uint
	addressDailyQuota uint
	tokenAmount       int64
}

// applicationRecord defines single record for verification.
type applicationRecord struct {
	rowID         int64
	applicationID string
	platform      string
	address       string
	mediaURL      string
	account       string
	state         State
	tokenAmount   int64 // covenantsql could store uint64 value, use int64 instead
	failReason    string
}

func (r *applicationRecord) asMap() (result map[string]interface{}) {
	result = make(map[string]interface{})

	result["rowID"] = r.rowID
	result["applicationID"] = r.applicationID
	result["platform"] = r.platform
	result["address"] = r.address
	result["mediaURL"] = r.mediaURL
	result["account"] = r.account
	result["state"] = r.state.String()
	result["tokenAmount"] = r.tokenAmount
	result["failReason"] = r.failReason

	return
}

// NewPersistence returns a new application persistence api.
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
				id string unique,
				platform string,
				account string, 
				url string,
				address string, 
				state int, 
				amount bigint, 
				reason string, 
				ctime datetime
			  )`)
	return
}

func (p *Persistence) checkAccountLimit(platform string, account string) (err error) {
	// TODO, consider cache the limits in memory?
	timeOfDayStart := time.Now().UTC().Format("2006-01-02 00:00:00")

	// account limit check
	row := p.db.QueryRowContext(context.Background(),
		`SELECT COUNT(1) AS cnt FROM faucet_records
		WHERE ctime >= ? AND platform = ? AND account = ? AND state IN (?, ?, ?)`,
		timeOfDayStart, platform, account, StateApplication, StateVerified, StateDispensed)

	var result uint

	err = row.Scan(&result)
	if err != nil {
		return
	}

	if result >= p.accountDailyQuota {
		// quota exceeded
		log.WithFields(log.Fields{
			"account":  account,
			"platform": platform,
		}).Error("daily account quota exceeded")
		return ErrAccountQuotaExceeded
	}

	return
}

func (p *Persistence) checkAddressLimit(address string) (err error) {
	// TODO, consider cache the limits in memory?
	timeOfDayStart := time.Now().UTC().Format("2006-01-02 00:00:00")

	// account limit check
	row := p.db.QueryRowContext(context.Background(),
		`SELECT COUNT(1) AS cnt FROM faucet_records
		WHERE ctime >= ? AND address = ? AND state IN (?, ?, ?)`,
		timeOfDayStart, address, StateApplication, StateVerified, StateDispensed)

	var result uint

	err = row.Scan(&result)
	if err != nil {
		return
	}

	if result >= p.addressDailyQuota {
		// quota exceeded
		log.WithFields(log.Fields{
			"address": address,
		}).Error("daily address quota exceeded")
		return ErrAddressQuotaExceeded
	}

	return
}

// enqueueApplication record a new token application to CovenantSQL database.
func (p *Persistence) enqueueApplication(address string, mediaURL string) (applicationID string, err error) {
	// resolve account name in address
	var meta urlMeta
	meta, err = extractPlatformInURL(mediaURL)
	if err != nil {
		log.WithFields(log.Fields{
			"address":  address,
			"mediaURL": mediaURL,
		}).Errorf("enqueue application with invalid url: %v", err)
		return
	}

	// check limits
	if err = p.checkAccountLimit(meta.platform, meta.account); err != nil {
		return
	}
	if err = p.checkAddressLimit(address); err != nil {
		return
	}

	// generate uuid
	applicationID = uuid.Must(uuid.NewV4()).String()

	// enqueue
	_, err = p.db.ExecContext(context.Background(),
		`INSERT INTO faucet_records (
				id,
				platform,
				account,
				url,
				address,
				state,
				amount,
				reason,
				ctime
			  ) VALUES (?, ?, ?, ?, ?, ?, ?, '', CURRENT_TIMESTAMP)`,
		applicationID, meta.platform, meta.account, mediaURL, address, StateApplication, p.tokenAmount)

	if err != nil {
		log.WithFields(log.Fields{
			"address":  address,
			"mediaURL": mediaURL,
		}).Errorf("enqueue application failed: %v", err)

		err = ErrEnqueueApplication
	}

	return
}

// queryState returns faucet application state.
func (p *Persistence) queryState(address string, applicationID string) (record *applicationRecord, err error) {
	row := p.db.QueryRowContext(context.Background(),
		`SELECT id, rowid, platform, address, url, account, state, amount, reason FROM faucet_records WHERE 
				address = ? AND id = ? LIMIT 1`, address, applicationID)

	record = &applicationRecord{}
	err = row.Scan(&record.applicationID, &record.rowID, &record.platform, &record.address, &record.mediaURL,
		&record.account, &record.state, &record.tokenAmount, &record.failReason)

	return
}

// getRecords fetch records need to be processed.
func (p *Persistence) getRecords(startRowID int64, platform string, state State, limitCount int) (records []*applicationRecord, err error) {
	var rows *sql.Rows

	args := make([]interface{}, 0)
	baseSQL := "SELECT id, rowid, platform, address, url, account, state, amount FROM faucet_records WHERE 1=1 "

	if startRowID > 0 {
		baseSQL += " AND rowid >= ? "
		args = append(args, startRowID)
	}
	if platform != "" {
		baseSQL += " AND platform = ? "
		args = append(args, platform)
	}
	if state != StateUnknown {
		baseSQL += " AND state = ? "
		args = append(args, state)
	}
	if limitCount > 0 {
		baseSQL += " LIMIT ?"
		args = append(args, limitCount)
	}

	rows, err = p.db.QueryContext(context.Background(), baseSQL, args...)

	for rows.Next() {
		r := &applicationRecord{}

		if err = rows.Scan(&r.applicationID, &r.rowID, &r.platform, &r.address, &r.mediaURL,
			&r.account, &r.state, &r.tokenAmount); err != nil {
			return
		}

		records = append(records, r)
	}

	return
}

// updateRecord updates application record.
func (p *Persistence) updateRecord(record *applicationRecord) (err error) {
	_, err = p.db.ExecContext(context.Background(),
		`UPDATE faucet_records SET
				id = ?,
				platform = ?,
				address = ?,
				url = ?,
				account = ?,
				state = ?,
				reason = ?,
				amount = ?
			  WHERE rowid = ?`,
		record.applicationID,
		record.platform,
		record.address,
		record.mediaURL,
		record.account,
		int(record.state),
		record.failReason,
		record.tokenAmount,
		record.rowID,
	)

	return
}
