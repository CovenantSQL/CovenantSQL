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

func initChainDBConnection(dbFile string) error {
	dsn := fmt.Sprintf("%s?_journal=WAL&mode=ro", dbFile)
	underdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return errors.WithMessage(err, "unable to open chain.db")
	}
	chaindb = &gorp.DbMap{
		Db:      underdb,
		Dialect: gorp.SqliteDialect{},
	}

	// register tables
	chaindb.AddTableWithName(Block{}, "indexed_blocks").SetKeys(false, "Height")
	chaindb.AddTableWithName(Transaction{}, "indexed_transactions").SetKeys(false, "BlockHeight", "TxIndex")

	return nil
}
