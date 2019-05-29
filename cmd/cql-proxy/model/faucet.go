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
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-proxy/utils"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	"gopkg.in/gorp.v1"
	"time"
)

type TokenApply struct {
	ID        string               `db:"id,"`
	Account   utils.AccountAddress `db:"account"`
	Developer int64                `db:"developer_id"`
	Amount    uint64               `db:"amount"`
	Created   int64                `db:"created"`
}

func CheckTokenApplyLimits(c *gin.Context, developer int64, account utils.AccountAddress, userLimits int64, accountLimits int64) (err error) {
	beginOfTheDay := time.Now().UTC().Truncate(24 * time.Hour).Unix()
	dbMap := c.MustGet(keyDB).(*gorp.DbMap)

	recordCount, err := dbMap.SelectInt(`SELECT COUNT(1) AS "cnt" FROM "token_apply" WHERE "created" >= ? AND "developer_id" = ?`,
		beginOfTheDay, developer)
	if err != nil {
		return
	}

	if recordCount >= userLimits {
		err = errors.New("user quota exceeded")
		return
	}

	recordCount, err = dbMap.SelectInt(`SELECT COUNT(1) AS "cnt" FROM "token_apply" WHERE "created" >= ? AND "account" = ?`,
		beginOfTheDay, account)
	if err != nil {
		return
	}

	if recordCount >= accountLimits {
		err = errors.New("account quota exceeded")
		return
	}

	return
}

func AddTokenApplyRecord(c *gin.Context, developer int64, account utils.AccountAddress, amount uint64) (r *TokenApply, err error) {
	applicationID := uuid.Must(uuid.NewV4()).String()

	r = &TokenApply{
		ID:        applicationID,
		Account:   account,
		Developer: developer,
		Amount:    amount,
		Created:   time.Now().Unix(),
	}

	err = c.MustGet(keyDB).(*gorp.DbMap).Insert(r)
	return
}

func init() {
	RegisterModel("token_apply", TokenApply{}, "ID", false)
}
