/*
 * Copyright 2016 The Cockroach Authors.
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package client

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/pkg/errors"
)

// ExecuteTx starts a transaction, and runs fn in it
func ExecuteTx(
	ctx context.Context, db *sql.DB, txopts *sql.TxOptions, fn func(*sql.Tx) error,
) error {
	// Start a transaction.
	tx, err := db.BeginTx(ctx, txopts)
	if err != nil {
		return err
	}
	return ExecuteInTx(tx, func() error { return fn(tx) })
}

// ExecuteInTx runs fn inside tx which should already have begun.
func ExecuteInTx(tx driver.Tx, fn func() error) (err error) {
	err = fn()
	if err == nil {
		err = tx.Commit()
		if err != nil {
			err = errors.Wrapf(err, "exec in tx")
		}
	} else {
		_ = tx.Rollback()
	}
	return
}
