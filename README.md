<img src="logo/covenantsql_horizontal.png" width=350>

[![Go Report Card](https://goreportcard.com/badge/github.com/CovenantSQL/CovenantSQL?style=flat-square)](https://goreportcard.com/report/github.com/CovenantSQL/CovenantSQL)
[![Coverage](https://codecov.io/gh/CovenantSQL/CovenantSQL/branch/develop/graph/badge.svg)](https://codecov.io/gh/CovenantSQL/CovenantSQL)
[![Build Status](https://travis-ci.org/CovenantSQL/CovenantSQL.png?branch=develop)](https://travis-ci.org/CovenantSQL/CovenantSQL)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/CovenantSQL/CovenantSQL)

## What is CovenantSQL?

CovenantSQL is a decentralized, history immutable, and crowdsourcing database that supports DApps on blockchain and also traditional Apps. CovenantSQL aims to build a high performance infrastructure which allows users to have full control over their data. The migration cost is minimized by providing standard database driver and SQL API so that developers could migrate to CovenantSQL easily.

Traditional databases run on expensive servers hosted
by IDCs and must be maintained by experienced engineers and DBAs. Also, high availability and scalability are expensive to achieve. CovenantSQL is a paid database service that addresses these problems using a sharing economy with the following features:

- **Decentralized**: database replication on the blockchain is controlled automatically by an algorithm.
- **Immutable**: all database changes are permanently recorded on the blockchain.
- **Secure**: all data storage and transfers are encrypted using [ETLS]((https://github.com/CovenantSQL/research/wiki/ETLS(Enhanced-Transport-Layer-Security))).
- **Open**: you control who your data is shared with by granting permissions. You have the option of making your database public-readable.
- **High Availability**: databases are replicated and distributed over the Internet. You can specify the number of replications.

## Mining

Miners in the CovenantSQL network are paid for offering computing resources. Miners with higher performance and stability receive more rewards.

## Installation

**CovenantSQL is Still Under Construction**
🚧🚧🚧🚧👷👷👷👷👷🚧🚧🚧🚧


### Requirements

CovenantSQL requires `Go` 1.10+. To install `Go`, follow this [link](https://golang.org/doc/install). 

In addition, [dep](https://github.com/golang/dep) is required to manage dependencies. 

### Getting the source

Clone the CovenantSQL repo:

```
git clone https://github.com/CovenantSQL/CovenantSQL.git
cd CovenantSQL
```

Install dependencies:
(*Note that to make `dep` work, you should put CovenantSQL's source code at proper position under your `$GOPATH`.*)

```
dep ensure -v
```

## API
- The source code is periodically indexed: [CovenantSQL API](https://godoc.org/github.com/CovenantSQL/CovenantSQL)

## Key Technologies Explaination

- Crypto
  - [ETLS Explaination](https://github.com/CovenantSQL/research/wiki/ETLS(Enhanced-Transport-Layer-Security))

- P2P Technology
  - [S/Kademlia](https://github.com/CovenantSQL/research/wiki/Secure-Kademlia)

- Consensus
  - [BFT-DPoS](https://github.com/CovenantSQL/research/wiki/BFT-DPoS)
  - PoE (Proof of Execution)
  - PoS (Proof of Storage)

- Zero-Knowledge Proof
  - [zk-SNARKS](https://github.com/CovenantSQL/research/wiki/zk-SNARKS)


## Support

- [mail us](mailto:webmaster@covenantsql.io)
- [submit issue](https://github.com/CovenantSQL/CovenantSQL/issues/new)



