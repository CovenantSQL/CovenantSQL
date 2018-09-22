idminer is a helper command of CovenantSQL

## Install 

Make sure that `$GOPATH/bin` is in your `$PATH`

```bash
$ go get github.com/CovenantSQL/CovenantSQL/cmd/idminer
```

Show usage of `idminer`:

```bash
$ idminer -help
```

## Usage
### Generate Key Pair

```
$ idminer -tool keygen
Enter master key(press Enter for default: ""): 
⏎
Private key file: private.key
Public key's hex: 03bc9e90e3301a2f5ae52bfa1f9e033cde81b6b6e7188b11831562bf5847bff4c0
```

The private.key is your encrypted private key file, and the pubkey hex is your public key's hex.

### Generate Wallet Address from existing Key

```
$ idminer -tool addrgen -private private.key
Enter master key(default: ""):
⏎
wallet address: 4jXvNvPHKNPU8Sncz5u5F5WSGcgXmzC1g8RuAXTCJzLsbF9Dsf9
$ idminer -tool addrgen -public 02f2707c1c6955a9019cd9d02ade37b931fbfa286a1163dfc1de965ec01a5c4ff8
wallet address: 4jXvNvPHKNPU8Sncz5u5F5WSGcgXmzC1g8RuAXTCJzLsbF9Dsf9
```

You can generate your *wallet* address for test net according to your private key or public key.
