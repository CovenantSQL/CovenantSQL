# Changelog

## [v0.0.1](https://github.com/CovenantSQL/CovenantSQL/tree/v0.0.1) (2018-09-27)

[Full Changelog](https://github.com/CovenantSQL/CovenantSQL/compare/82811a8fcac65d74aefbb506450e4477ecdad048...v0.0.1)

**TestNet**
 
1. Ready for CLI or SDK usage. For now, Linux & OSX supported only.
1. SQL Chain Explorer is ready.

**TestNet Known Issues**

1. Main Chain
   1. Allocation algorithm for BlockProducer and Miner is incomplete.
   1. Joining as BP or Miner is unsupported for now. _Fix@2018-10-12_
   1. Forking Recovery algorithm is incomplete.
1. Connector
   1. [Java](https://github.com/CovenantSQL/covenant-connector) and [Golang Connector](https://github.com/CovenantSQL/CovenantSQL/tree/develop/client) is ready.
   1. ĐApp support for ETH or EOS is incomplete. 
   1. Java connector protocol is based on RESTful HTTPS, change to Golang DH-RPC latter.
1. Database
   1. Cartesian product or big join caused OOM. _Fix@2018-10-12_
   1. SQL Query filter is incomplete. _Fix@2018-10-12_
   1. Forking Recovery algorithm is incomplete.
   1. Database for TestNet is World Open on [Explorer](https://explorer.dbhub.org).

**Closed issues:**

- ThunderDB has been renamed to CovenantSQL [\#58](https://github.com/CovenantSQL/CovenantSQL/issues/58)
- build error [\#50](https://github.com/CovenantSQL/CovenantSQL/issues/50)

**Merged pull requests:**

- Make idminer and README.md less ambiguous [\#77](https://github.com/CovenantSQL/CovenantSQL/pull/77) ([auxten](https://github.com/auxten))
- Make all path config in adapter relative to working root configuration [\#76](https://github.com/CovenantSQL/CovenantSQL/pull/76) ([auxten](https://github.com/auxten))
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
- Add observer helper functions and binaries and hotfix binary for dht database migrations. [\#47](https://github.com/CovenantSQL/CovenantSQL/pull/47) ([xq262144](https://github.com/xq262144))
- [More...](https://github.com/CovenantSQL/CovenantSQL/pulls?q=is%3Apr+is%3Aclosed)



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
