This doc introduce the usage of covenantSQL `cli`. `cli` is a command line interface for batch scripting used for creating, querying, updating, and deleting the SQLChain and database adhere to the SQLChain.

## Install
Download [Latest Release](https://github.com/CovenantSQL/CovenantSQL/releases) or build from src:

```bash
$ go get github.com/CovenantSQL/CovenantSQL/client
$ go get github.com/CovenantSQL/CovenantSQL/cmd/cli
```
*make sure that `$GOPATH/bin` is in your `$PATH`*

### Generating Default Config File

You need to provide a config and a master key for initialization. The master key is used to encrypt/decrypt local key pair. If you generate a config file with `idminer`, you can find the config file in the directory that `idminer` create.

See: [idminer doc](https://github.com/CovenantSQL/CovenantSQL/tree/develop/cmd/idminer#usage) for conf generation and get wallet address.

### Check balance

Use `covenantcli` to check your wallet balance:
```bash
$ ./covenantcli -config conf/config.yaml -get-balance
INFO[0000] 
### Public Key ###
0388954cf083bb6bb2b9c7248849b57c76326296fcc0d69764fc61eedb5b8d820c
### Public Key ###
  caller="privatekeystore.go:116 crypto/kms.InitLocalKeyPair"
INFO[0000] stable coin balance is: 100                   caller="main.go:246 main.main"
INFO[0000] covenant coin balance is: 0                   caller="main.go:247 main.main"
```
Here, I got **"stable coin balance is: 100"**.

## Initialize a CovenantSQL `cli`

After you prepare your master key and config file, CovenantSQL `cli` can be initialized by:
You can get a database id when create a new SQL Chain:

```bash
$ covenantcli -config conf/config.yaml -create 1
INFO[0000]
### Public Key ###
039bc931161383c994ab9b81e95ddc1494b0efeb1cb735bb91e1043a1d6b98ebfd
### Public Key ###
  caller="privatekeystore.go:116 crypto/kms.InitLocalKeyPair"
INFO[0000] the newly created database is: covenantsql://0e9103318821b027f35b96c4fd5562683543276b72c488966d616bfe0fe4d213  caller="main.go:297 main.main"
```

Here, `-create 1` refers that there is only one node in SQL Chain.

```bash
$ covenantcli -config conf/config.yaml -dsn covenantsql://address
```
`address` is database id. 

Show the complete usage of `covenantcli`:

```bash
$ covenantcli -help
```

## Use the `cli`

Free to use the `cli` now:

```bash
co:address=> show tables;
```
