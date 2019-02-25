# Changelog

## [v0.3.0](https://github.com/CovenantSQL/CovenantSQL/tree/v0.3.0) (2019-01-30)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/v0.2.0...v0.3.0)

**Closed issues:**

- Blocks are not written to chain [\#219](https://github.com/CovenantSQL/CovenantSQL/issues/219)

**Merged pull requests:**

- Improve database query performance [\#240](https://github.com/CovenantSQL/CovenantSQL/pull/240) ([xq262144](https://github.com/xq262144))
- Support query regulations and flag bit permissions [\#239](https://github.com/CovenantSQL/CovenantSQL/pull/239) ([xq262144](https://github.com/xq262144))
- Run each round sequentially to decrease running goroutines [\#238](https://github.com/CovenantSQL/CovenantSQL/pull/238) ([leventeliu](https://github.com/leventeliu))
- Fix bug: bad critical section for multiple values [\#237](https://github.com/CovenantSQL/CovenantSQL/pull/237) ([leventeliu](https://github.com/leventeliu))
- Add missing private key and rename apinode to fullnode [\#236](https://github.com/CovenantSQL/CovenantSQL/pull/236) ([ggicci](https://github.com/ggicci))
- Regen HashStablePack for v2.0.0 [\#235](https://github.com/CovenantSQL/CovenantSQL/pull/235) ([auxten](https://github.com/auxten))
- Use ~/.cql/ directory as default config location. [\#233](https://github.com/CovenantSQL/CovenantSQL/pull/233) ([laodouya](https://github.com/laodouya))
- GetCurrentBP also return BP follower [\#229](https://github.com/CovenantSQL/CovenantSQL/pull/229) ([auxten](https://github.com/auxten))
- Use 114 DNS for default [\#228](https://github.com/CovenantSQL/CovenantSQL/pull/228) ([auxten](https://github.com/auxten))
-  Add metric web for cqld and cql-minerd [\#227](https://github.com/CovenantSQL/CovenantSQL/pull/227) ([auxten](https://github.com/auxten))
- Add testnet client init process test. Add a param 'fast' for GNTE test [\#226](https://github.com/CovenantSQL/CovenantSQL/pull/226) ([laodouya](https://github.com/laodouya))
-  Fix bug to avoid ack DDoS and add timeout for connecting db [\#225](https://github.com/CovenantSQL/CovenantSQL/pull/225) ([zeqing-guo](https://github.com/zeqing-guo))
-  Add readonly flag for fuse [\#224](https://github.com/CovenantSQL/CovenantSQL/pull/224) ([auxten](https://github.com/auxten))
- Add other cmd tools in observer image [\#222](https://github.com/CovenantSQL/CovenantSQL/pull/222) ([zeqing-guo](https://github.com/zeqing-guo))
- Add cql-utils option to wait for confirmation [\#221](https://github.com/CovenantSQL/CovenantSQL/pull/221) ([leventeliu](https://github.com/leventeliu))
- Add isolation level for xenomint state [\#220](https://github.com/CovenantSQL/CovenantSQL/pull/220) ([leventeliu](https://github.com/leventeliu))
- Add TransactionState MarshalHash [\#218](https://github.com/CovenantSQL/CovenantSQL/pull/218) ([auxten](https://github.com/auxten))
- Fix block producer genesis block hash mismatch [\#217](https://github.com/CovenantSQL/CovenantSQL/pull/217) ([leventeliu](https://github.com/leventeliu))
- Fix gitlab ci script pipline will not return failed when go test failed. [\#216](https://github.com/CovenantSQL/CovenantSQL/pull/216) ([laodouya](https://github.com/laodouya))
- Add query payload encode cache [\#215](https://github.com/CovenantSQL/CovenantSQL/pull/215) ([auxten](https://github.com/auxten))
- Client log optimize [\#214](https://github.com/CovenantSQL/CovenantSQL/pull/214) ([auxten](https://github.com/auxten))
- Add testnet compatibility test in CI process. [\#212](https://github.com/CovenantSQL/CovenantSQL/pull/212) ([laodouya](https://github.com/laodouya))
- Fix block producers forking on startup [\#211](https://github.com/CovenantSQL/CovenantSQL/pull/211) ([leventeliu](https://github.com/leventeliu))
- Coping with sqlchain soft forks [\#201](https://github.com/CovenantSQL/CovenantSQL/pull/201) ([xq262144](https://github.com/xq262144))
- Support JSON RPC API [\#164](https://github.com/CovenantSQL/CovenantSQL/pull/164) ([ggicci](https://github.com/ggicci))

## [v0.2.0](https://github.com/CovenantSQL/CovenantSQL/tree/v0.2.0) (2019-01-05)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/v0.1.0...v0.2.0)

**Merged pull requests:**

- Update GNTE config [\#193](https://github.com/CovenantSQL/CovenantSQL/pull/193) ([laodouya](https://github.com/laodouya))
- Fix matchProvidersWithUser inconsistent [\#188](https://github.com/CovenantSQL/CovenantSQL/pull/188) ([auxten](https://github.com/auxten))
- Speed up BPs at genesis startup [\#186](https://github.com/CovenantSQL/CovenantSQL/pull/186) ([leventeliu](https://github.com/leventeliu))
- Wait for database creation fix [\#185](https://github.com/CovenantSQL/CovenantSQL/pull/185) ([xq262144](https://github.com/xq262144))
- Simplify cql and cql-utils log [\#184](https://github.com/CovenantSQL/CovenantSQL/pull/184) ([auxten](https://github.com/auxten))
- Fix Makefile PHONY, add push\_testnet [\#183](https://github.com/CovenantSQL/CovenantSQL/pull/183) ([auxten](https://github.com/auxten))
- Fix issue: duplicate branches [\#182](https://github.com/CovenantSQL/CovenantSQL/pull/182) ([leventeliu](https://github.com/leventeliu))
- Update testnet conf [\#181](https://github.com/CovenantSQL/CovenantSQL/pull/181) ([auxten](https://github.com/auxten))
- Remove base58 wallet address [\#179](https://github.com/CovenantSQL/CovenantSQL/pull/179) ([auxten](https://github.com/auxten))
- Fix GNTE test config missing miner wallet init coin [\#178](https://github.com/CovenantSQL/CovenantSQL/pull/178) ([laodouya](https://github.com/laodouya))
- Upgrade transaction structure: add Timestamp field [\#177](https://github.com/CovenantSQL/CovenantSQL/pull/177) ([ggicci](https://github.com/ggicci))
- Block main cycle when BP network is unreachable [\#176](https://github.com/CovenantSQL/CovenantSQL/pull/176) ([leventeliu](https://github.com/leventeliu))
- Remove useless hash in base58 encoded private key [\#175](https://github.com/CovenantSQL/CovenantSQL/pull/175) ([auxten](https://github.com/auxten))
- Prune unused codes [\#174](https://github.com/CovenantSQL/CovenantSQL/pull/174) ([leventeliu](https://github.com/leventeliu))
- Fix docker entry point [\#173](https://github.com/CovenantSQL/CovenantSQL/pull/173) ([leventeliu](https://github.com/leventeliu))
- Add permission granting/revoking [\#172](https://github.com/CovenantSQL/CovenantSQL/pull/172) ([leventeliu](https://github.com/leventeliu))
- Extract observer to an independent docker image [\#163](https://github.com/CovenantSQL/CovenantSQL/pull/163) ([laodouya](https://github.com/laodouya))

## [v0.1.0](https://github.com/CovenantSQL/CovenantSQL/tree/v0.1.0) (2018-12-29)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/v0.0.6...v0.1.0)

**Closed issues:**

- Private Key Format should be human readable [\#144](https://github.com/CovenantSQL/CovenantSQL/issues/144)

**Merged pull requests:**

- Change test config IsTestMode to true. [\#171](https://github.com/CovenantSQL/CovenantSQL/pull/171) ([laodouya](https://github.com/laodouya))
- Update node\_c config for testnet [\#170](https://github.com/CovenantSQL/CovenantSQL/pull/170) ([leventeliu](https://github.com/leventeliu))
- Fix miner crash on billing [\#169](https://github.com/CovenantSQL/CovenantSQL/pull/169) ([leventeliu](https://github.com/leventeliu))
- Update ci config [\#168](https://github.com/CovenantSQL/CovenantSQL/pull/168) ([xq262144](https://github.com/xq262144))
- Update observer api to support queries pagination [\#167](https://github.com/CovenantSQL/CovenantSQL/pull/167) ([xq262144](https://github.com/xq262144))
-  Add testnet parameters package and fix cql-utils congen tool [\#166](https://github.com/CovenantSQL/CovenantSQL/pull/166) ([leventeliu](https://github.com/leventeliu))
- Update ci config, run reviewdog on travis, other in gitlab [\#165](https://github.com/CovenantSQL/CovenantSQL/pull/165) ([xq262144](https://github.com/xq262144))
-  Add README-zh for cql-utils  [\#161](https://github.com/CovenantSQL/CovenantSQL/pull/161) ([leventeliu](https://github.com/leventeliu))
- Update client readme and example [\#160](https://github.com/CovenantSQL/CovenantSQL/pull/160) ([laodouya](https://github.com/laodouya))
- Add more test cases for ETLS [\#159](https://github.com/CovenantSQL/CovenantSQL/pull/159) ([auxten](https://github.com/auxten))
- Reduce unnecessary object copy while producing/applying new block [\#158](https://github.com/CovenantSQL/CovenantSQL/pull/158) ([leventeliu](https://github.com/leventeliu))
- HTTP\(S\) Adapter Improvements and various query sanitizations [\#157](https://github.com/CovenantSQL/CovenantSQL/pull/157) ([xq262144](https://github.com/xq262144))
- Add raw socket magic header and encrypted magic header for ETLS [\#156](https://github.com/CovenantSQL/CovenantSQL/pull/156) ([auxten](https://github.com/auxten))
- Fix RunCommandNB pipe issue [\#155](https://github.com/CovenantSQL/CovenantSQL/pull/155) ([auxten](https://github.com/auxten))
- Fix some issues in block producer [\#154](https://github.com/CovenantSQL/CovenantSQL/pull/154) ([leventeliu](https://github.com/leventeliu))
- Use docker mapping port for node\_c [\#150](https://github.com/CovenantSQL/CovenantSQL/pull/150) ([auxten](https://github.com/auxten))
- Update default makefile task to all [\#147](https://github.com/CovenantSQL/CovenantSQL/pull/147) ([draveness](https://github.com/draveness))
- Save & load private key in base58 format [\#146](https://github.com/CovenantSQL/CovenantSQL/pull/146) ([draveness](https://github.com/draveness))
- Add billing process and chain bus support [\#145](https://github.com/CovenantSQL/CovenantSQL/pull/145) ([zeqing-guo](https://github.com/zeqing-guo))
- Refactor build.sh and Makefile [\#142](https://github.com/CovenantSQL/CovenantSQL/pull/142) ([laodouya](https://github.com/laodouya))
- Block producer refactor and chain bus integration [\#135](https://github.com/CovenantSQL/CovenantSQL/pull/135) ([leventeliu](https://github.com/leventeliu))

## [v0.0.6](https://github.com/CovenantSQL/CovenantSQL/tree/v0.0.6) (2018-12-18)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/v0.0.5...v0.0.6)

**Closed issues:**

- Could not run in docker based alpine image [\#134](https://github.com/CovenantSQL/CovenantSQL/issues/134)
- Quickstart cli document 404 [\#97](https://github.com/CovenantSQL/CovenantSQL/issues/97)
- Any plan to NodeJS support? [\#80](https://github.com/CovenantSQL/CovenantSQL/issues/80)

**Merged pull requests:**

- Add a Gitter chat badge to README.md [\#137](https://github.com/CovenantSQL/CovenantSQL/pull/137) ([gitter-badger](https://github.com/gitter-badger))
- Add DSN options to enable SQL queries on follower nodes [\#136](https://github.com/CovenantSQL/CovenantSQL/pull/136) ([ggicci](https://github.com/ggicci))
- If smux session dead, cancel the context passed to RPC through Envelope [\#133](https://github.com/CovenantSQL/CovenantSQL/pull/133) ([auxten](https://github.com/auxten))
- Add new cloudflare.com DNSKEY [\#132](https://github.com/CovenantSQL/CovenantSQL/pull/132) ([auxten](https://github.com/auxten))
- Prepare for auto bench in jenkins environment. [\#131](https://github.com/CovenantSQL/CovenantSQL/pull/131) ([laodouya](https://github.com/laodouya))

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
