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
        <img src="https://platform.twitter.com/widgets/follow_button.b510f289fb017e5dfdc7fdb287a0ae4b.en.html#dnt=false&id=twitter-widget-2&lang=en&screen_name=CovenantLabs&show_count=false&show_screen_name=true&size=m"
            alt="follow on Twitter"></a>
</p>


[![Go Report Card](https://goreportcard.com/badge/github.com/CovenantSQL/CovenantSQL?style=flat-square)](https://goreportcard.com/report/github.com/CovenantSQL/CovenantSQL)
[![Coverage](https://codecov.io/gh/CovenantSQL/CovenantSQL/branch/develop/graph/badge.svg)](https://codecov.io/gh/CovenantSQL/CovenantSQL)
[![Build Status](https://travis-ci.org/CovenantSQL/CovenantSQL.png?branch=develop)](https://travis-ci.org/CovenantSQL/CovenantSQL)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/CovenantSQL/CovenantSQL)

## What is CovenantSQL?

CovenantSQL is a decentralized, crowdsourcing SQL database on blockchain. with Features:

- **SQL**: most SQL-92 support.
- **Decentralize**: decentralize with our consensus algorithm DH-RPC & Kayak.
- **Privacy**: access with granted permission and Encryption Pass.
- **Immutable**: query history in CovenantSQL is immutable and trackable.

## Howto

#### 1 line makes App to ĐApp
```go
sql.Open("CovenantSQL", dbURI)
```


But CovenantSQL is Still Under Construction(U know..). Test net will be released till Oct. 

Star or follow us [@CovenantLabs](https://twitter.com/CovenantLabs), we will *Wake U Up When September Ends*

## Why

[On the next Internet, everyone should have a complete **Data Rights**](https://medium.com/@covenant_labs/covenantsql-the-sql-database-on-blockchain-db027aaf1e0e)

## Architecture

![CovenantSQL 3 Layer design](logo/arch.png)

1. **Global Consensus Layer** (the main chain, the middle ring in the architecture diagram):
    - There will only be one main chain throughout the network.
    - Mainly responsible for database Miner and the user’s contract matching, transaction settlement, anti-cheating, shard chain lock hash and other global consensus matters.
1. **SQL Consensus Layer** (shard chain, rings on both sides):
    - Each database will have its own separate shard chain.
    - Mainly responsible for: the signature, delivery and consistency of the various Transactions of the database. The data history of the permanent traceability is mainly implemented here, and the hash lock is performed in the main chain.
1. **Datastore Layer** (database engine with SQL-92 support):
    - Each Database has its own independent distributed engine.
    - Mainly responsible for: database storage & encryption, query processing & signature, efficient indexing.


## Technically

#### Network Stack

<img src="logo/DH-RPC-Layer.png" width=350>

  - [DH-RPC](rpc/) = TLS - Cert + DHT.
    - [**E**nhanced **TLS**](https://github.com/CovenantSQL/research/wiki/ETLS(Enhanced-Transport-Layer-Security)): the Transport Layer Security.

  
#### Test Tools
  -  [(**G**lobal **N**etwork **T**opology **E**mulator)](https://github.com/CovenantSQL/GNTE) is used for network emulating.


#### Connector

- [Golang](client/)
- [Java](https://github.com/CovenantSQL/covenant-connector)
- Coding for more……

## Contact

- [mail us](mailto:webmaster@covenantsql.io)
- [submit issue](https://github.com/CovenantSQL/CovenantSQL/issues/new)
- <a href="https://twitter.com/intent/follow?screen_name=CovenantLabs">
          <img src="https://platform.twitter.com/widgets/follow_button.b510f289fb017e5dfdc7fdb287a0ae4b.en.html#dnt=false&id=twitter-widget-2&lang=en&screen_name=CovenantLabs&show_count=false&show_screen_name=true&size=m"
              alt="follow on Twitter"></a>



