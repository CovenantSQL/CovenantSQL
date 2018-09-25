This doc introduce the usage of covenantSQL `cli`. `cli` is a command line interface for batch scripting used for creating, querying, updating, and deleting the SQLChain and database adhere to the SQLChain.

## Prerequisites

Make sure that `$GOPATH/bin` is in your `$PATH`

```bash
$ go get github.com/CovenantSQL/CovenantSQL/client
$ go get github.com/CovenantSQL/CovenantSQL/cmd/cli
```

### Generating Default Config File

```bash
$ idminer -tool confgen -difficulty 8 -root bp -testnet
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

## Initialize a CovenantSQL `cli`

You need to provide a config and a master key for initialization. The master key is used to encrypt/decrypt local key pair. If you generate a config file with `idminer`, you can find the config file in the directory that `idminer` create.

After you prepare your master key and config file, CovenantSQL `cli` can be initialized by:

```bash
$ covenantcli --config config_file
```

## Use the `cli`

Free to use the `cli` now:

```bash
\c covenantsql://address
conenantsql://address> sqlite instructions
```

Note: you can get SQL Chain address when you create a SQL Chain.
