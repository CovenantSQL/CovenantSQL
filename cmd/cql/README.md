This doc introduces the usage of CovenantSQL command line client `cql`. `cql` is a command line interface for batch scripting used for creating, querying, updating, and deleting the SQLChain and database adhere to the SQLChain.

## Install
Download [Latest Release](https://github.com/CovenantSQL/CovenantSQL/releases) or build from src:

```bash
$ go get github.com/CovenantSQL/CovenantSQL/client
$ go get github.com/CovenantSQL/CovenantSQL/cmd/cql
```
*make sure that `$GOPATH/bin` is in your `$PATH`*

## Generating Default Config File

You need to provide a config and a master key for initialization. The master key is used to encrypt/decrypt local key pair. If you generate a config file with `cql generate config`, you can find the config file in the directory `~/.cql`.

```
$ cql generate config
Enter master key(press Enter for default: ""):
⏎
Private key file: private.key
Public key's hex: 03bc9e90e3301a2f5ae52bfa1f9e033cde81b6b6e7188b11831562bf5847bff4c0
```

The ~/.cql/private.key is your encrypted private key file, and the pubkey hex is your public key's hex.

### Generate Wallet Address from existing Key

```
$ cql wallet
Enter master key(default: ""):
⏎
wallet address: 4jXvNvPHKNPU8Sncz5u5F5WSGcgXmzC1g8RuAXTCJzLsbF9Dsf9
$ cql generate -config ~/.cql/config.yaml wallet
Enter master key(default: ""):
⏎
wallet address: 4jXvNvPHKNPU8Sncz5u5F5WSGcgXmzC1g8RuAXTCJzLsbF9Dsf9
```

You can generate your *wallet* address for test net according to your private key(default ~/.cql/private).

## Check balance

Use `cql` to check your wallet balance:

```bash
$ cql wallet -balance all
INFO[0000] 
### Public Key ###
0388954cf083bb6bb2b9c7248849b57c76326296fcc0d69764fc61eedb5b8d820c
### Public Key ###
  caller="privatekeystore.go:116 crypto/kms.InitLocalKeyPair"
INFO[0000] stable coin balance is: 100                   caller="main.go:246 main.main"
INFO[0000] covenant coin balance is: 0                   caller="main.go:247 main.main"
```
Here, I got **"stable coin balance is: 100"**.

## Initialize a CovenantSQL `cql`

After you prepare your master key and config file, CovenantSQL `cql` can be initialized by:
You can get a database id when create a new SQL Chain:

```bash
# if a non-default password applied on master key, use `-password` to pass it
$ cql create '{"node":1}'
INFO[0000]
### Public Key ###
039bc931161383c994ab9b81e95ddc1494b0efeb1cb735bb91e1043a1d6b98ebfd
### Public Key ###
  caller="privatekeystore.go:116 crypto/kms.InitLocalKeyPair"
INFO[0000] the newly created database is: covenantsql://0e9103318821b027f35b96c4fd5562683543276b72c488966d616bfe0fe4d213  caller="main.go:297 main.main"
```

Here, `create '{"node":1}'` refers that there is only one node in SQL Chain.

```bash
$ cql console -dsn covenantsql://address
```
`address` is database id. 

Show the complete usage of `cql`:

```bash
$ cql help
```

## Use the `cql`

Free to use the `cql` now:

```bash
co:address=> show tables;
```
