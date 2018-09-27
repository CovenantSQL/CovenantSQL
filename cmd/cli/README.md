This doc introduce the usage of covenantSQL `cli`. `cli` is a command line interface for batch scripting used for creating, querying, updating, and deleting the SQLChain and database adhere to the SQLChain.

## Install
Download [Latest Release](https://github.com/CovenantSQL/CovenantSQL/releases) or build from src:

```bash
$ go get github.com/CovenantSQL/CovenantSQL/client
$ go get github.com/CovenantSQL/CovenantSQL/cmd/cli
```
*make sure that `$GOPATH/bin` is in your `$PATH`*

### Generating Default Config File

```bash
$ idminer -tool confgen -root conf
Generating key pair...
Enter master key(press Enter for default: ""):
⏎
Private key file: conf/private.key
Public key's hex: 02296ea73240dcd69d2b3f1fb754c8debdf68c62147488abb10165428667ec8cbd
Generated key pair.
Generating nonce...
nonce: {{731613648 0 0 0} 11 001ea9c8381c4e8bb875372df9e02cd74326cbec33ef6f5d4c6829fcbf5012e9}
node id: 001ea9c8381c4e8bb875372df9e02cd74326cbec33ef6f5d4c6829fcbf5012e9
Generated nonce.
Generating config file...
Generated nonce.
```

Then, you can find private key and config.yaml in conf.

## Initialize a CovenantSQL `cli`

You need to provide a config and a master key for initialization. The master key is used to encrypt/decrypt local key pair. If you generate a config file with `idminer`, you can find the config file in the directory that `idminer` create.

After you prepare your master key and config file, CovenantSQL `cli` can be initialized by:

```bash
$ covenantcli -config conf/config.yaml -dsn covenantsql://address
```

`address` is database id. You can get a database id when create a new SQL Chain:

```bash
$ covenantcli -config conf/config.yaml -create 1
INFO[0000]
### Public Key ###
039bc931161383c994ab9b81e95ddc1494b0efeb1cb735bb91e1043a1d6b98ebfd
### Public Key ###
  caller="privatekeystore.go:116 crypto/kms.InitLocalKeyPair"
INFO[0000] the newly created database is: covenantsql://0e9103318821b027f35b96c4fd5562683543276b72c488966d616bfe0fe4d213  caller="main.go:297 main.main"
```

Here, `1` refers that there is only one node in SQL Chain.

Show the complete usage of `covenantcli`:

```bash
$ covenantcli -help
```

## Use the `cli`

Free to use the `cli` now:

```bash
co:address=> show tables;
```
