# Changelog

## [v0.0.5](https://github.com/CovenantSQL/CovenantSQL/tree/v0.0.5) (2018-11-23)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/v0.0.4...v0.0.5)

**Fixed bugs:**

- Stuck in 2pc inconsistent state error [\#56](https://github.com/CovenantSQL/CovenantSQL/issues/56)

**Closed issues:**

- 用 cql 来查看你的钱包余额时出错 [\#111](https://github.com/CovenantSQL/CovenantSQL/issues/111)

**Merged pull requests:**

- Fix table name should add space in one test case. [\#128](https://github.com/CovenantSQL/CovenantSQL/pull/128) ([laodouya](https://github.com/laodouya))
- Fix memory exhausting issue [\#127](https://github.com/CovenantSQL/CovenantSQL/pull/127) ([leventeliu](https://github.com/leventeliu))
- Add block cache pruning [\#126](https://github.com/CovenantSQL/CovenantSQL/pull/126) ([leventeliu](https://github.com/leventeliu))
- Utils/Profiler log field name wrong. [\#124](https://github.com/CovenantSQL/CovenantSQL/pull/124) ([laodouya](https://github.com/laodouya))
- Add simple Pub Sub framework and fix bug during long march [\#122](https://github.com/CovenantSQL/CovenantSQL/pull/122) ([auxten](https://github.com/auxten))
- Move client.conn.pCaller init in newConn [\#121](https://github.com/CovenantSQL/CovenantSQL/pull/121) ([auxten](https://github.com/auxten))
- Fix broken BenchmarkMinerXXX add BenchmarkMinerTwo to travis [\#120](https://github.com/CovenantSQL/CovenantSQL/pull/120) ([auxten](https://github.com/auxten))
- FUSE on CovenantSQL [\#119](https://github.com/CovenantSQL/CovenantSQL/pull/119) ([auxten](https://github.com/auxten))
- Integration bench test support on exist database file. [\#118](https://github.com/CovenantSQL/CovenantSQL/pull/118) ([laodouya](https://github.com/laodouya))
- Move HashSignVerifier definition to crypto package [\#117](https://github.com/CovenantSQL/CovenantSQL/pull/117) ([leventeliu](https://github.com/leventeliu))
- Increase project test coverage and fix bugs in kayak [\#116](https://github.com/CovenantSQL/CovenantSQL/pull/116) ([xq262144](https://github.com/xq262144))
- Fix invalid parent, and increate block producing period on main chain [\#115](https://github.com/CovenantSQL/CovenantSQL/pull/115) ([zeqing-guo](https://github.com/zeqing-guo))
- A shard chain eventual consistency implementation [\#103](https://github.com/CovenantSQL/CovenantSQL/pull/103) ([leventeliu](https://github.com/leventeliu))

## [v0.0.4](https://github.com/CovenantSQL/CovenantSQL/tree/v0.0.4) (2018-11-08)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/v0.0.3...v0.0.4)

**Fixed bugs:**

- Potential deadlock in testing [\#93](https://github.com/CovenantSQL/CovenantSQL/issues/93)

**Closed issues:**

- Where can I find covenantsql.io/covenantsql\_adapter [\#53](https://github.com/CovenantSQL/CovenantSQL/issues/53)

**Merged pull requests:**

- Fix loadChain failure, remove the lock in sync\(\) [\#114](https://github.com/CovenantSQL/CovenantSQL/pull/114) ([zeqing-guo](https://github.com/zeqing-guo))
- Kayak performance improvement refactor [\#112](https://github.com/CovenantSQL/CovenantSQL/pull/112) ([xq262144](https://github.com/xq262144))
- Fix index out of bound, refactor part of sqlchain code [\#110](https://github.com/CovenantSQL/CovenantSQL/pull/110) ([leventeliu](https://github.com/leventeliu))
- Support lastInsertID/affectedRows in kayak [\#109](https://github.com/CovenantSQL/CovenantSQL/pull/109) ([xq262144](https://github.com/xq262144))

## [v0.0.3](https://github.com/CovenantSQL/CovenantSQL/tree/v0.0.3) (2018-11-04)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/v0.0.2...v0.0.3)

**Fixed bugs:**

- Cannot receive tokens from testnet [\#84](https://github.com/CovenantSQL/CovenantSQL/issues/84)

**Closed issues:**

- Command cqld -version failed without -config parameter [\#99](https://github.com/CovenantSQL/CovenantSQL/issues/99)

**Merged pull requests:**

- Add call stack print for Error, Fatal and Panic [\#108](https://github.com/CovenantSQL/CovenantSQL/pull/108) ([auxten](https://github.com/auxten))
- Update GNTE submodule to it's newest master [\#105](https://github.com/CovenantSQL/CovenantSQL/pull/105) ([laodouya](https://github.com/laodouya))
- Add GNTE bench test [\#102](https://github.com/CovenantSQL/CovenantSQL/pull/102) ([laodouya](https://github.com/laodouya))
- Use leveldb to instead boltdb on sqlchain [\#101](https://github.com/CovenantSQL/CovenantSQL/pull/101) ([zeqing-guo](https://github.com/zeqing-guo))
- Update Dockerfile, using go1.11 instead [\#100](https://github.com/CovenantSQL/CovenantSQL/pull/100) ([laodouya](https://github.com/laodouya))
- Replace hashicorp/yamux with xtaci/smux [\#98](https://github.com/CovenantSQL/CovenantSQL/pull/98) ([auxten](https://github.com/auxten))
- Update MySQL Adapter to support mysql-java-connector/SequelPro/Navicat [\#96](https://github.com/CovenantSQL/CovenantSQL/pull/96) ([xq262144](https://github.com/xq262144))
- Minerd performance tuning [\#95](https://github.com/CovenantSQL/CovenantSQL/pull/95) ([auxten](https://github.com/auxten))
- Update README [\#91](https://github.com/CovenantSQL/CovenantSQL/pull/91) ([foreseaz](https://github.com/foreseaz))
- Blockproducer Explorer feature including Transaction type encode/decode improvements [\#90](https://github.com/CovenantSQL/CovenantSQL/pull/90) ([xq262144](https://github.com/xq262144))

## [v0.0.2](https://github.com/CovenantSQL/CovenantSQL/tree/v0.0.2) (2018-10-17)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/v0.0.1...v0.0.2)

**Closed issues:**

- Improve commit messages for better project tracking and changelogs [\#62](https://github.com/CovenantSQL/CovenantSQL/issues/62)

**Merged pull requests:**

- Use c implementation of secp256k1 in Sign and Verify [\#89](https://github.com/CovenantSQL/CovenantSQL/pull/89) ([auxten](https://github.com/auxten))
- Provide a runnable MySQL adapter using mysql text protocol [\#87](https://github.com/CovenantSQL/CovenantSQL/pull/87) ([xq262144](https://github.com/xq262144))
- Sanitize SQL query before applying to underlying storage engine [\#85](https://github.com/CovenantSQL/CovenantSQL/pull/85) ([xq262144](https://github.com/xq262144))
- Limit codecov threshold to 0.5% [\#83](https://github.com/CovenantSQL/CovenantSQL/pull/83) ([auxten](https://github.com/auxten))
- Add SQLite and 1, 2, 3 miner\(s\) with or without signature benchmark test suits [\#82](https://github.com/CovenantSQL/CovenantSQL/pull/82) ([auxten](https://github.com/auxten))
- Fix a fatal bug while querying ACK from other peer [\#81](https://github.com/CovenantSQL/CovenantSQL/pull/81) ([leventeliu](https://github.com/leventeliu))
- Fix fetch block API issue [\#79](https://github.com/CovenantSQL/CovenantSQL/pull/79) ([leventeliu](https://github.com/leventeliu))
- Fix observer dynamic subscribe from oldest [\#78](https://github.com/CovenantSQL/CovenantSQL/pull/78) ([auxten](https://github.com/auxten))

## [v0.0.1](https://github.com/CovenantSQL/CovenantSQL/tree/v0.0.1) (2018-09-27)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/82811a8fcac65d74aefbb506450e4477ecdad048...v0.0.1)

**Closed issues:**

- ThunderDB has been renamed to CovenantSQL [\#58](https://github.com/CovenantSQL/CovenantSQL/issues/58)
- build error [\#50](https://github.com/CovenantSQL/CovenantSQL/issues/50)

**Merged pull requests:**

- Make idminer and README.md less ambiguous [\#77](https://github.com/CovenantSQL/CovenantSQL/pull/77) ([auxten](https://github.com/auxten))
- Make all path config in adapter relative to working root configuration [\#76](https://github.com/CovenantSQL/CovenantSQL/pull/76) ([auxten](https://github.com/auxten))
- TestNet faucet for CovenantSQL API demo [\#75](https://github.com/CovenantSQL/CovenantSQL/pull/75) ([auxten](https://github.com/auxten))
- HTTPS RESTful API for CovenantSQL [\#74](https://github.com/CovenantSQL/CovenantSQL/pull/74) ([xq262144](https://github.com/xq262144))
- Unify the nonce increment, BaseAccount also increases account nonce. [\#73](https://github.com/CovenantSQL/CovenantSQL/pull/73) ([leventeliu](https://github.com/leventeliu))
- Fix a nonce checking issue and add more specific test cases. [\#72](https://github.com/CovenantSQL/CovenantSQL/pull/72) ([leventeliu](https://github.com/leventeliu))
- Add an idminer readme for generating key pair and testnet address [\#71](https://github.com/CovenantSQL/CovenantSQL/pull/71) ([zeqing-guo](https://github.com/zeqing-guo))
- Add BenchmarkSingleMiner, use rpc.NewPersistentCaller for client conn [\#70](https://github.com/CovenantSQL/CovenantSQL/pull/70) ([auxten](https://github.com/auxten))
- Add RPC methods for balance query. [\#69](https://github.com/CovenantSQL/CovenantSQL/pull/69) ([leventeliu](https://github.com/leventeliu))
- Addrgen to generate testnet address [\#68](https://github.com/CovenantSQL/CovenantSQL/pull/68) ([zeqing-guo](https://github.com/zeqing-guo))
- Support hole skipping in observer [\#67](https://github.com/CovenantSQL/CovenantSQL/pull/67) ([xq262144](https://github.com/xq262144))
- Add base account type transaction with initial balance for testnet. [\#66](https://github.com/CovenantSQL/CovenantSQL/pull/66) ([leventeliu](https://github.com/leventeliu))
- Add auto config generator [\#65](https://github.com/CovenantSQL/CovenantSQL/pull/65) ([zeqing-guo](https://github.com/zeqing-guo))
- Use a well-defined interface to process transactions on block producers. [\#64](https://github.com/CovenantSQL/CovenantSQL/pull/64) ([leventeliu](https://github.com/leventeliu))
- Add an explanation for non-deterministic authenticated encryption input vector [\#63](https://github.com/CovenantSQL/CovenantSQL/pull/63) ([auxten](https://github.com/auxten))
- Add nonce generator in idminer [\#61](https://github.com/CovenantSQL/CovenantSQL/pull/61) ([zeqing-guo](https://github.com/zeqing-guo))
- Optional database encryption support on database creation. [\#60](https://github.com/CovenantSQL/CovenantSQL/pull/60) ([xq262144](https://github.com/xq262144))
- Clarify README.md for project and DH-RPC [\#59](https://github.com/CovenantSQL/CovenantSQL/pull/59) ([auxten](https://github.com/auxten))
- Rename ThunderDB to CovenantSQL [\#57](https://github.com/CovenantSQL/CovenantSQL/pull/57) ([zeqing-guo](https://github.com/zeqing-guo))
- Update README.md [\#55](https://github.com/CovenantSQL/CovenantSQL/pull/55) ([auxten](https://github.com/auxten))
- Add address test cases [\#54](https://github.com/CovenantSQL/CovenantSQL/pull/54) ([zeqing-guo](https://github.com/zeqing-guo))
- Fix block index issue [\#52](https://github.com/CovenantSQL/CovenantSQL/pull/52) ([leventeliu](https://github.com/leventeliu))
- Fix/issue-50: use major version tag in docker file [\#51](https://github.com/CovenantSQL/CovenantSQL/pull/51) ([leventeliu](https://github.com/leventeliu))
- Add DH-RPC example [\#49](https://github.com/CovenantSQL/CovenantSQL/pull/49) ([auxten](https://github.com/auxten))
- Merge cli tool to core code base [\#48](https://github.com/CovenantSQL/CovenantSQL/pull/48) ([xq262144](https://github.com/xq262144))



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
