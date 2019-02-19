<p align="center">
    <img src="logo/covenantsql_horizontal.png"
        height="130">
</p>
<p align="center">
    <a href="https://goreportcard.com/report/github.com/CovenantSQL/CovenantSQL">
        <img src="https://goreportcard.com/badge/github.com/CovenantSQL/CovenantSQL?style=flat-square"
            alt="Go Report Card"></a>
    <a href="https://codecov.io/gh/CovenantSQL/CovenantSQL">
        <img src="https://codecov.io/gh/CovenantSQL/CovenantSQL/branch/develop/graph/badge.svg"
            alt="Coverage"></a>
    <a href="https://travis-ci.org/CovenantSQL/CovenantSQL">
        <img src="https://travis-ci.org/CovenantSQL/CovenantSQL.png?branch=develop"
            alt="Build Status"/></a>
    <a href="https://opensource.org/licenses/Apache-2.0">
        <img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg"
            alt="License"></a>
    <a href="https://godoc.org/github.com/CovenantSQL/CovenantSQL">
        <img src="https://img.shields.io/badge/godoc-reference-blue.svg"
            alt="GoDoc"></a>
    <a href="https://twitter.com/intent/follow?screen_name=CovenantLabs">
        <img src="https://img.shields.io/twitter/url/https/twitter.com/fold_left.svg?style=social&label=Follow%20%40CovenantLabs"
            alt="follow on Twitter"></a>
    <a href="https://gitter.im/CovenantSQL/CovenantSQL?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge">
        <img src="https://badges.gitter.im/CovenantSQL/CovenantSQL.svg"
            alt="Join the chat at https://gitter.im/CovenantSQL/CovenantSQL"></a>
</p>

[中文简介](https://github.com/CovenantSQL/CovenantSQL/blob/develop/README-zh.md)

CovenantSQL is a decentralized, crowdsourcing SQL database on blockchain with features:

- **SQL**: most SQL-92 support.
- **Decentralize**: decentralize with our consensus algorithm DH-RPC & Kayak.
- **Privacy**: access with granted permission and Encryption Pass.
- **Immutable**: query history in CovenantSQL is immutable and trackable.

We believe [On the next Internet, everyone should have a complete **Data Rights**](https://medium.com/@covenant_labs/covenantsql-the-sql-database-on-blockchain-db027aaf1e0e)

#### One Line Makes Data on Blockchain
```go
sql.Open("CovenantSQL", dbURI)
```

## 

![CovenantSQL 3 Layer design](logo/arch.png)

- Layer 1: **Global Consensus Layer** (the main chain, the middle ring in the architecture diagram):
    - There will only be one main chain throughout the network.
    - Mainly responsible for database Miner and the user’s contract matching, transaction settlement, anti-cheating, shard chain lock hash and other global consensus matters.
- Layer 2: **SQL Consensus Layer** (shard chain, rings on both sides):
    - Each database will have its own separate shard chain.
    - Mainly responsible for: the signature, delivery and consistency of the various Transactions of the database. The data history of the permanent traceability is mainly implemented here, and the hash lock is performed in the main chain.
- Layer 3: **Datastore Layer** (database engine with SQL-92 support):
    - Each Database has its own independent distributed engine.
    - Mainly responsible for: database storage & encryption, query processing & signature, efficient indexing.

## Papers
Our team members published:

- [Thunder crystal: a novel crowdsourcing-based content distribution platform](https://dl.acm.org/citation.cfm?id=2736085)
- [Analyzing streaming performance in crowdsourcing-based video service systems](https://ieeexplore.ieee.org/abstract/document/7114727/)
- [Performance Analysis of Thunder Crystal: A Crowdsourcing-Based Video Distribution Platform](https://ieeexplore.ieee.org/abstract/document/7762143/)

that inspired us:

- [Bitcoin: A Peer-to-Peer Electronic Cash System](https://bitcoin.org/bitcoin.pdf)
- [S/Kademlia](https://github.com/thunderdb/research/wiki/Secure-Kademlia)
    - [S/Kademlia: A practicable approach towards secure key-based routing](https://ieeexplore.ieee.org/document/4447808/)
- [vSQL: Verifying arbitrary SQL queries over dynamic outsourced databases](https://ieeexplore.ieee.org/abstract/document/7958614/)

## Libs

### Network Stack

[DH-RPC](rpc/) := TLS - Cert + DHT

| Layer              | Implementation |
|:-------------------|:--------------:|
| RPC                |     `net/rpc`    |
| Naming             |      [**C**onsistent **S**ecure **DHT**](https://godoc.org/github.com/CovenantSQL/CovenantSQL/consistent)     |
| Pooling            |  Session Pool  |
| Multiplex          |      [smux](https://github.com/xtaci/smux)     |
| Transport Security |      [**E**nhanced **TLS**](https://github.com/CovenantSQL/research/wiki/ETLS(Enhanced-Transport-Layer-Security))      |
| Network            |       TCP or KCP for optional later      |


#### Test Tools
  - [**G**lobal **N**etwork **T**opology **E**mulator](https://github.com/CovenantSQL/GNTE) is used for network emulating.
  - [Liner Consistency Test](https://github.com/anishathalye/porcupine)


#### Connector

CovenantSQL is still under construction and Testnet is already released, [have a try](https://developers.covenantsql.io/docs/quickstart).


- [Golang](client/)
- [Java](https://github.com/CovenantSQL/covenant-connector)
- [NodeJS](https://github.com/CovenantSQL/node-covenantsql)
- [Python](https://github.com/CovenantSQL/python-driver)
- Coding for more……

Watch us or [![follow on Twitter](https://img.shields.io/twitter/url/https/twitter.com/fold_left.svg?style=social&label=Follow%20%40CovenantLabs)](https://twitter.com/intent/follow?screen_name=CovenantLabs) for updates.

## TestNet

- [Quick Start](https://developers.covenantsql.io)

## Contact

- [mail us](mailto:webmaster@covenantsql.io)
- <a href="https://twitter.com/intent/follow?screen_name=CovenantLabs">
          <img src="https://img.shields.io/twitter/url/https/twitter.com/fold_left.svg?style=social&label=Follow%20%40CovenantLabs"
              alt="follow on Twitter"></a>

- [![Join the chat at https://gitter.im/CovenantSQL/CovenantSQL](https://badges.gitter.im/CovenantSQL/CovenantSQL.svg)](https://gitter.im/CovenantSQL/CovenantSQL?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
