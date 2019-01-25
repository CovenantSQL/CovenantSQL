package models

import (
	"database/sql"
	"fmt"

	_ "github.com/CovenantSQL/go-sqlite3-encrypt" // sqlite3 driver
	"github.com/go-gorp/gorp"
	"github.com/pkg/errors"
)

var (
	chaindb *gorp.DbMap
)

// InitModels setup the models package.
func InitModels(dbFile string) error {
	return initChainDBConnection(dbFile)
}

// OpenSQLiteDBAsGorp opens a sqlite database an wrapped it in gorp.DbMap.
func OpenSQLiteDBAsGorp(dbFile, mode string, maxOpen, maxIdle int) (db *gorp.DbMap, err error) {
	dsn := fmt.Sprintf("%s?_journal=WAL&mode=%s", dbFile, mode)
	underdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open database %q", dsn)
	}
	underdb.SetMaxOpenConns(maxOpen)
	underdb.SetMaxIdleConns(maxIdle)

	if err := underdb.Ping(); err != nil {
		return nil, errors.Wrapf(err, "ping to database %q failed", dsn)
	}

	db = &gorp.DbMap{
		Db:      underdb,
		Dialect: gorp.SqliteDialect{},
	}
	return db, nil
}

func initChainDBConnection(dbFile string) (err error) {
	chaindb, err = OpenSQLiteDBAsGorp(dbFile, "ro", 100, 30)
	if err != nil {
		return err
	}

	// register tables
	chaindb.AddTableWithName(Block{}, "indexed_blocks").SetKeys(false, "Height")
	chaindb.AddTableWithName(Transaction{}, "indexed_transactions").SetKeys(false, "BlockHeight", "TxIndex")
	chaindb.AddTableWithName(Account{}, "accounts").SetKeys(false, "Address")

	return nil
}
