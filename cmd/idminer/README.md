## Install `idminer`

Make sure that `$GOPATH/bin` is in your `$PATH`

```shell
go get github.com/CovenantSQL/CovenantSQL/cmd/idminer
```

The usage of `idminer`:

```
> idminer -help
Usage of ./idminer:
  -addrgen
        addrgen generates a testnet address from your key pair
  -config string
        rpc config file
  -difficulty int
        difficulty for miner to mine nodes and generating nonce (default 256)
  -endpoint string
        rpc endpoint to do test call
  -private string
        private key file to generate/show
  -public string
        public key hex string to mine node id/nonce
  -req string
        rpc request to do test call, in json format
  -root string
        confgen root is the working root directory containing all auto-generating keys and certifications (default "node")
  -rpc string
        rpc name to do test call
  -testnet
        use confgen with testnet will download the testnet certification from our testnet
  -tool string
        tool type, miner, keygen, keytool, rpc, nonce, confgen, addrgen (default "miner")
```

## Generate Key Pair

```
> idminer -tool keygen -private private.key
Enter master key(default: ""):

public key's hex: 03dfd8b7dda0a9e8b4530ab9d145c4dafa377619bded878928807a0e2a08059a2e
```

The private.key is your encrypted private key file, and the pubkey hex is your public key's hex.

## Generate Wallet Address from Private/Public Key

```
> idminer -tool addrgen -private private.key
Enter master key(default: ""):

test net address: 4jXvNvPHKNPU8Sncz5u5F5WSGcgXmzC1g8RuAXTCJzLsbF9Dsf9
> idminer -tool addrgen --public 02f2707c1c6955a9019cd9d02ade37b931fbfa286a1163dfc1de965ec01a5c4ff8
test net address: 4jXvNvPHKNPU8Sncz5u5F5WSGcgXmzC1g8RuAXTCJzLsbF9Dsf9
```

You can generate your *wallet* address for test net according to your private key or public key.
