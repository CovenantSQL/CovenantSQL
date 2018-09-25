This doc introduce the usage of covenantSQL client. Client is used for creating, querying, updating, and deleting the SQLChain and database adhere to the SQLChain.

## Prerequisites

Make sure that `$GOPATH/bin` is in your `$PATH`

```bash
$ go get github.com/CovenantSQL/CovenantSQL/client
$ go get github.com/CovenantSQL/CovenantSQL/cmd/idminer
```

and import `client` package if you want to use it in your code.

### Generating Default Config File

```bash
$ idminer -tool confgen -root bp
Generating key pair...
Enter master key(press Enter for default: ""):
⏎
Private key file: bp/private.key
Public key's hex: 02296ea73240dcd69d2b3f1fb754c8debdf68c62147488abb10165428667ec8cbd
Generated key pair.
Generating nonce...
nonce: {{731613648 0 0 0} 11 001ea9c8381c4e8bb875372df9e02cd74326cbec33ef6f5d4c6829fcbf5012e9}
node id: 001ea9c8381c4e8bb875372df9e02cd74326cbec33ef6f5d4c6829fcbf5012e9
Generated nonce.
Generating config file...
Generated nonce.
```

## Initialize a CovenantSQL Client

You need to provide a config and a master key for initialization. The master key is used to encrypt/decrypt local key pair. If you generate a config file with `idminer`, you can find the config file in the directory that `idminer` create.

After you prepare your master key and config file, CovenantSQL client can be initialized by:

```golang
client.Init(configFile, masterKey)
```

## Client Usage

### Create a SQLChain Database

To create a new SQL Chain, the number of node should be provided:

```golang
var dsn string
dsn, err := client.ResourceMeta{Node: uint16(nodeCnt)}
// process err
var cfg *client.Config
cfg, err = client.ParseDSN(dsn)
// process err
```

Database ID can be found in `cfg`:

```golang
databaseID := cfg.DatabaseID
```

In all:

```golang
func Create(nodeCnt uint16) (dbID string, err error) {
	var dsn string
	if dsn, err = client.Create(client.ResourceMeta{Node: uint16(nodeCnt)}); err != nil {
		return
	}

	var cfg *client.Config
	if cfg, err = client.ParseDSN(dsn); err != nil {
		return
	}

	dbID = cfg.DatabaseID
	return
}
```

### Query and Exec

When you get the database ID, you can query or execute some sql on SQL Chain as follows:

```golang
func Query(dbID string, query string) (result , err error) {
	var conn *sql.DB
	if conn, err = s.getConn(dbID); err != nil {
		return
	}
	defer conn.Close()

	var rows *sql.Rows
	if rows, err = conn.Query(query); err != nil {
		return
	}
	defer rows.Close()

	// read the rows of rows
}

func Exec(dbID string, query string) (err error) {
	var conn *sql.DB
	if conn, err = s.getConn(dbID); err != nil {
		return
	}
	defer conn.Close()

	_, err = conn.Exec(query)

	return
}

func getConn(dbID string) (db *sql.DB, err error) {
	cfg := client.NewConfig()
	cfg.DatabaseID = dbID

	return sql.Open("covenantsql", cfg.FormatDSN())
}
```

### Drop the Database

Drop your database on SQL Chain is very easy with your database ID:

```golang
func Drop(dbID string) (err error) {
	cfg := client.NewConfig()
	cfg.DatabaseID = dbID
	err = client.Drop(cfg.FormatDSN())
	return
}
```

