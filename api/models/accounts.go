package models

// AccountsModel groups operations on Accounts.
type AccountsModel struct{}

// Account represents an account.
type Account struct {
	Address      string `db:"address" json:"address"`
	EncodedBytes []byte `db:"encoded" json:"-"`
	// TODO: add more fields
}

// CountAccounts counts the accounts.
func (m *AccountsModel) CountAccounts() (count int64, err error) {
	querySQL := `SELECT COUNT(*) FROM "accounts";`
	return chaindb.SelectInt(querySQL)
}
