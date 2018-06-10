<img src="logo/logo_blue.png" width=350>

[![Go Report Card](https://goreportcard.com/badge/github.com/thunderdb/ThunderDB?style=flat-square)](https://goreportcard.com/report/github.com/thunderdb/ThunderDB)
[![Coverage](https://codecov.io/gh/thunderdb/ThunderDB/branch/develop/graph/badge.svg)](https://codecov.io/gh/thunderdb/ThunderDB)
[![Build Status](https://travis-ci.org/thunderdb/ThunderDB.png?branch=develop)](https://travis-ci.org/thunderdb/ThunderDB)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/thunderdb/ThunderDB)

## What is ThunderDB?

ThunderDB is a distributed database running on BlockChain for ĐApps and traditional Apps. 

Traditional databases run on expensive servers hosted
by IDCs and must be maintained by experienced engineers and DBAs. Also, high availability and scalability are expensive to achieve. ThunderDB is a paid database service that addresses these problems using a sharing economy with the following features:

- Decentralized: database replication on the blockchain is controlled automatically by an algorithm.
- Secure: all data storage and transfers are encrypted using [ETLS]((https://github.com/thunderdb/research/wiki/ETLS(Enhanced-Transport-Layer-Security))).
- Open: you control who your data is shared with by granting permissions. You have the option of making your database public-readable.
- High Availability: databases are replicated and distributed over the Internet. You can specify the number of replications.
- Immutable: all database changes are permanently recorded on the blockchain.

## Mining

Miners in the ThunderDB network are paid for offering computing resources. Miners with higher performance and stability receive more rewards.

## Installation

**ThunderDB is Still Under Heavy Construction**
🚧🚧🚧🚧👷👷👷👷👷🚧🚧🚧🚧


### Requirements

ThunderDB requires `Go` 1.10+. To install `Go`, follow this [link](https://golang.org/doc/install). 

In addition, [dep](https://github.com/golang/dep) is required to manage dependencies. 

### Getting the source

Clone the ThunderDB repo:

```
git clone https://github.com/thunderdb/ThunderDB.git
cd ThunderDB
```

Install dependencies:
(*Note that to make `dep` work, you should put ThunderDB's source code at proper position under your `$GOPATH`.*)

```
dep ensure -v
```

## API
- https://godoc.org/github.com/thunderdb/ThunderDB

## Key Technologies Explaination

#### Crypto

- [ETLS Explaination](https://github.com/thunderdb/research/wiki/ETLS(Enhanced-Transport-Layer-Security))

#### P2P Technology

- [S/Kademlia](https://github.com/thunderdb/research/wiki/Secure-Kademlia)

#### Consensus

- [BFT-DPoS](https://github.com/thunderdb/research/wiki/BFT-DPoS)
- PoE (Proof of Execution)
- PoS (Proof of Storage)

#### Zero-Knowledge Proof

- [zk-SNARKS](https://github.com/thunderdb/research/wiki/zk-SNARKS)


## Support

- [Our Mail](mailto:webmaster@thunderdb.io)



