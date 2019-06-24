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

package main

import (
	"encoding/json"
	"time"

	gorp "gopkg.in/gorp.v2"
)

type AuditRecord struct {
	ID      int64       `db:"id"`
	Hash    string      `db:"hash"`
	Time    int64       `db:"time"`
	Op      string      `db:"op"`
	RawData []byte      `db:"data"`
	Data    interface{} `db:"-"`
	Error   string      `db:"error"`
}

func (r *AuditRecord) PostGet(gorp.SqlExecutor) error {
	return r.Deserialize()
}

func (r *AuditRecord) PreUpdate(gorp.SqlExecutor) error {
	return r.Serialize()
}

func (r *AuditRecord) PreInsert(gorp.SqlExecutor) error {
	return r.Serialize()
}

func (r *AuditRecord) Serialize() (err error) {
	r.RawData, err = json.Marshal(r.Data)
	return
}

func (r *AuditRecord) Deserialize() (err error) {
	err = json.Unmarshal(r.RawData, &r.Data)
	return
}

func AddAuditRecord(db *gorp.DbMap, r *AuditRecord) (err error) {
	r.Time = time.Now().Unix()
	err = db.Insert(r)
	return
}

func FindAuditRecords(db *gorp.DbMap, h string, fromTime int64, toTime int64, offset int64, limit int64) (err error) {
	return
}
