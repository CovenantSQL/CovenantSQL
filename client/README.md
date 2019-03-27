This doc introduce the usage of CovenantSQL client. Client is used for creating, querying, updating, and deleting the SQLChain and database adhere to the SQLChain.

## Prerequisites

Make sure that `$GOPATH/bin` is in your `$PATH`

```bash
$ go get github.com/CovenantSQL/CovenantSQL/client
$ go get github.com/CovenantSQL/CovenantSQL/cmd/cql
```

and import `client` package if you want to use it in your code.


## Initialize a CovenantSQL Client

You need to provide a config and a master key for initialization. The master key is used to encrypt/decrypt local key pair. Here is how to generate a default config with master key

### Generating Default Config File

Run `cql` like below. Enter a master key (like a password) for generating local key pair. After that, it may take a few seconds with a private key file and config.yaml file generated in `~/.cql/` folder.

```bash
$ cql generate config
Generating key pair...
Enter master key(press Enter for default: ""):
⏎
Private key file: ~/.cql/private.key
Public key's hex: 025abec9b0072615170f4acf4a2fa1162a13864bb66bc3f140b29f6bf50ceafc75
Generated key pair.
Generating nonce...
INFO[0005] cpu: 1
INFO[0005] position: 0, shift: 0x0, i: 0
nonce: {{1450338416 0 0 0} 26 0000002dd8bdb50ba0270642e4c4bc593c1630ef7784653f311b3c3d6374e514}
node id: 0000002dd8bdb50ba0270642e4c4bc593c1630ef7784653f311b3c3d6374e514
Generated nonce.
Generating config file...
Generated config.
```

After you prepare your master key and config file, CovenantSQL client can be initialized by:

```go
client.Init(configFile, masterKey)
```

## Client Usage

### Create a SQLChain Database

To create a new SQL Chain, the number of node should be provided:

```go
var (
	dsn string
	meta client.ResourceMeta
)
meta.Node = uint16(nodeCount)
dsn, err = client.Create(meta)
// process err
```
And you will get a dsn string. It represents a database instance, use it for queries.

### Query and Exec

When you get the dsn, you can query or execute some sql on SQL Chain as follows:

```go

	db, err := sql.Open("covenantsql", dsn)
	// process err

	_, err = db.Exec("CREATE TABLE testSimple ( column int );")
	// process err

	_, err = db.Exec("INSERT INTO testSimple VALUES(?);", 42)
	// process err

	row := db.QueryRow("SELECT column FROM testSimple LIMIT 1;")

	var result int
	err = row.Scan(&result)
	// process err
	fmt.Printf("SELECT column FROM testSimple LIMIT 1; result %d\n", result)

	err = db.Close()
	// process err

```
It just like other standard go sql database.

### Drop the Database

Drop your database on SQL Chain is very easy with your dsn string:

```go
	err = client.Drop(dsn)
	// process err
```

### Full Example

simple and complex client examples can be found in [client/_example](_example/)
