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

## Why

It’s safer to put data on an offline computer, but it’s also easy to accidentally lost and not easy to check. Whether it is Facebook or WeChat, various cloud disks, the user’s data is almost always stored in a database controlled by a large company. The data is yours and the big Internet company’s, but in the end it is still under the control of the big Internet company, and you have to confirm the terms of use.

The various data on the Internet can be roughly divided into two categories:

- Personal Data
  - For example: personal identity & account information; private property; personally published content; historical data on applications, websites.
  - Status: Privacy breaches, data mining abuse, and digital copyright infringement.
  - Wish: Everyone should have control over the reading and modification of personal data, as well as the rights to profit and authorization.
- Public Data
  - For example: Wikipedia and other co-create works; knowledge of human civilization; various data shared by the author to everyone.
  - Status: Public wiki is sometimes malicious tampering lose credibility；papers, documents are held hostage to a profit; Producers of valuable data have no benefits, and sometimes even the authorship cannot be guaranteed.
  - Wish: The production and accumulation of knowledge is a process of common creation, and each person’s contribution should be recorded.
  
Richard in “Silicon Valley” describes a "decentralized Internet". The "decentralized Internet" requires a decentralized database. The traditional database encountered a blockchain, and the data `Insert`, `Update`, and `Delete` became `Append`. "Append instead of overwrite" allows the history of the data to be fully recorded.

In order to change the status, there is a long way to go. A decentralized database provides at least the possibility for users to control their own data.

  > on the next generation of Internet, everyone should have a complete **Data Rights**

To give a simple example: In the future, our personal data can be stored in a decentralized cloud database. Like Bitcoin, we can completely control our data with a single key. We can develop a standard similar to the PCI DSS of the credit card industry. We call it the **GDSS (General Data Security Standard)** for the time being. The core is to require the manufacturer to strictly limit the use of the user’s data and delete it after use. For example, suppose Facebook is our trusted vendor. Following GDSS, we can give Facebook a key that we can only read our name, age, and friends list. At the same time, Facebook will record every time we read our data. If we find that Facebook has used our data to do something we don’t want, we can revoke this key and blame it at any time.

Another example: the biggest headache for startups or research organizations in the big data industry is that there is no data. Ordinary users can authorize these research institutions to use their own data and get a certain amount of compensation. This will create a win-win situation and avoid the data hegemony of the giants that store a lot of our data.
The EU’s GDPR is currently a very leading standard in this regard.

SEE: [Our Blog](https://medium.com/@covenant_labs/covenantsql-the-sql-database-on-blockchain-db027aaf1e0e)

## Architecture
Programmers familiar with the principles of distributed systems should know that from the perspective of CAP theorem, Blockchain is a final consistency algorithm. The PoW used by Bitcoin for the typical Blockchain 1.0 is mainly for dealing with non-trusted networks and nodes. From a popular point of view, almost we all have such an experience: “The less people involved, the more efficient the decision-making.”
Blockchain developers seem to be aware of the same problem. The new generation of Blockchain systems led by EOS uses a “cabinet” like DPoS. CovenantSQL was designed with this in mind at the beginning, using the layered architecture shown below:

![CovenantSQL 3 Layer design](logo/arch.png)

The architecture of CovenantSQL is mainly divided into three layers:

1. Global consensus layer (the main chain, the middle ring in the architecture diagram):
    - There will only be one main chain throughout the network.
    - Mainly responsible for database Miner and the user’s contract matching, transaction settlement, anti-cheating, side chain lock hash and other global consensus matters.
1. SQL consensus layer (sidechain, rings on both sides):
    - Each database will have its own separate sidechain.
    - Mainly responsible for: the signature, delivery and consistency of the various Transactions of the database. The data history of the permanent traceability is mainly implemented here, and the hash lock is performed in the main chain.
1. Database layer:
    - Each Database has its own independent distributed engine.
    - Mainly responsible for: database storage & encryption, query processing & signature, efficient indexing.

The three-layer design is hash locked with each other to ensure that the data cannot be falsified. From top to bottom, according to different requirements, different consensus algorithms and smaller consensus ranges are adopted to achieve higher consensus efficiency and performance.

The so-called non-tamperable blockchain is not 100% theoretically non-tamperable. The the economic cost of tampering is higher as the number of nodes increases. Different scenarios have different needs for data security, so in our design, the database creator decides the number of copies of the database. Of course, the more copies, the higher the cost. For a single Database, assuming that the number of instances required by the Database creator is `n`, then there will be no less than `n` nodes running the Database, and only the `n` nodes need to maintain the full data of the Database. The correspondence between Database and miner is anonymous to reduce the possibility of being attacked.

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

- DH-RPC
  - [DH-RPC is RPC framework powered CovenantSQL](rpc/)
  
- GNTE
  - [Global Network Topology Emulator](https://github.com/CovenantSQL/GNTE) is used to test CovenantSQL
  
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



