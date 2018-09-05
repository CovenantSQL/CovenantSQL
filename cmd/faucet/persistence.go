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
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
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
)

// Persistence defines the persistence api for faucet service.
type Persistence struct {
	db    *sql.Conn
	quota int
}

// NewPersistence returns a new application persistence api.
func NewPersistence() (p *Persistence, err error) {
	return
}

// EnqueueApplication record a new token application to CovenantSQL database.
func (p *Persistence) EnqueueApplication(address string, mediaURL string) (err error) {
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

	// check previous applications
	timeOfDayStart := time.Now().In(time.FixedZone("PRC", 8*60*60)).Format("2006-01-02 00:00:00")

	row := p.db.QueryRowContext(context.Background(),
		"SELECT COUNT(1) AS cnt FROM faucet_records WHERE ctime >= ? AND platform = ? AND account = ? AND address = ?",
		timeOfDayStart, meta.platform, meta.account, address)

	var result int

	err = row.Scan(&result)
	if err != nil {
		return
	}

	if result > p.quota {
		// quota exceeded
		log.WithFields(log.Fields{
			"address":  address,
			"mediaURL": mediaURL,
		}).Errorf("")
		return ErrQuotaExceeded
	}

	// enqueue
	_, err = p.db.ExecContext(context.Background(),
		"INSERT INTO faucet_records (platform, account, url, address, state) VALUES(?, ?, ?, ?)",
		meta.platform, meta.account, mediaURL, address, StateApplication)
	if err != nil {
		log.WithFields(log.Fields{
			"address":  address,
			"mediaURL": mediaURL,
		}).Errorf("enqueue application failed: %v", err)
		return ErrEnqueueApplication
	}

	return
}

// GetToVerifyRecords fetch records need to be processed.
func (p *Persistence) GetToVerifyRecords(startRowID int, platform string, limitCount int) (err error) {
	_, err = p.db.QueryContext(context.Background(),
		"SELECT rowid, * FROM faucet_records WHERE rowid >= ? AND platform = ? LIMIT ?",
		startRowID, platform, limitCount)

	return
}
