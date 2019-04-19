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

CovenantSQL(CQL) 是一个运行在公网上的 SQL 数据库，并具有 GDPR 合规与去中心化等特点。


- **GDPR 合规**: 无痛 GDPR 合规
- **SQL 接口**: 支持 SQL-92 标准
- **去中心化**: 基于共识算法 DH-RPC & Kayak 实现的去中心化
- **不可篡改**: CQL 中的 Query 历史是不可篡改且可追溯的
- **隐私**:  使用列级 ACL 和 SQL 白名单模式授予权限

我们坚信 [在下一个互联网时代，每个人都应该有完整的**数据权利**](https://medium.com/@covenant_labs/covenantsql-the-sql-database-on-blockchain-db027aaf1e0e)


**一行代码让数据上链**
```go
sql.Open("cql", dbURI)
```
## CQL 是什么?

- [Amazon QLDB](https://aws.amazon.com/qldb/)的开源版
- 比如 [filecoin](https://filecoin.io/) + [IPFS](https://ipfs.io/) 是去中心化的文件系统, CQL 就是去中心化的数据库

## CQL 如何工作

### 3 层架构


![CovenantSQL 3 Layer design](logo/arch.png)

- 第一层: **全局共识层**(主链，架构图中的中间环):
    - 整个网络中只有一个主链。
    - 主要负责数据库矿工与用户的合同匹配，交易结算，反作弊，子链哈希锁定等全局共识事宜。
- 第二层: **SQL 共识层**(子链，架构图中的两边环):
    - 每个数据库都有自己独立的子链。
    - 主要负责数据库各种事务的签名，交付和一致性。这里主要实现永久可追溯性的数据历史，并且在主链中执行哈希锁定。
- 第三层: **数据储存层**(支持 SQL-92 的数据库引擎):
    - 每个数据库都有自己独立的分布式引擎。
    - 主要负责：数据库存储和加密；查询处理和签名；高效索引。


### 共识算法

CQL 支持两种共识算法：

1. DPoS (委任权益证明) 应用在数据库的 `最终一致性` 和矿工所在的`第一层 (全局共识层)` ，CQL 的矿工在客户端将所有SQL查询及其签名打包成块，从而形成整个区块链，我们把这个算法命名为 [`Xenomint`](https://github.com/CovenantSQL/CovenantSQL/tree/develop/xenomint)
2. BFT-Raft (拜占庭容错算法)<sup>[bft-raft](#bft-raft)</sup> 应用于数据库的 `强一致性`。我们把这个实现命名为 [`Kayak`](https://github.com/CovenantSQL/CovenantSQL/tree/develop/kayak).  矿工 leader 会基于 `Kayak` 做 `两阶段提交` 来支持 `Transaction`.<sup>[transaction](#transaction)</sup>

在创建数据库的时候可以用命令行  `cql create '{"UseEventualConsistency": true, "Node": 3}'` 来计算 CQL 数据库的一致性节点


## 项目对比

|                              | 以太坊            | Hyperledger Fabric     | 亚马逊 QLDB | CovenantSQL                                                  |
| ---------------------------- | ----------------- | ---------------------- | ----------- | ------------------------------------------------------------ |
| **开发语言**                  | Solidity, ewasm   | Chaincode (Go, NodeJS) | ?           | Python, Golang, Java, PHP, NodeJS, MatLab                    |
| **开发模式**                  | Smart   Contract  | Chaincode              | SQL         | SQL                                                          |
| **是否开源**                  | Y                 | Y                      | N           | Y                                                            |
| **高可用节点**                | 3                 | 15                     | ?           | 3                                                            |
| **ACL 列查询**                | N                 | Y                      | ?           | Y                                                            |
| **数据格式**                  | File              | Key-value              | Document    | File<sup>[fuse](#fuse)</sup>, Key-value, Structured     |
| **存储加密**                  | N                 | API                    | Y           | Y                                                            |
| **数据脱敏**                  | N                 | N                      | N           | Y                                                            |
| **多个用户**                  | DIY               | DIY                    | N           | Y                                                            |
| **吞吐量（1秒延迟）**          | 15~10 tx/s        | 3500 tx/s              | ?           | 11065 tx/s (Eventually Consistency)<br/>1866 tx/s (Strong Consistency) |
| **一致性延迟**                | 2~6 min           | < 1 s                  | ?           | < 10 ms                                                      |
| **开放网络上的安全性**         | Y                 | N                      | Only in AWS | Y                                                            |
| **共识机制**                  | PoW + PoS(Casper) | CFT                    | ?           | DPoS (Eventually Consistency)<br/>BFT-Raft (Strong Consistency) |

#### 注释：
- <a name="bft-raft">BFT-Raft</a>: 一个 CQL leader 想要离线需要矿工来决定是否等待 leader 上线，以保证数据的完整性，或者提拔 follwers 以保证数据可用性，这部分仍在迭代中，欢迎任何建议。

- <a name="transaction">Transaction</a>: 说到 `ACID`，CQL 具有完整的“一致性，隔离性，耐用性”和有限的 `Atomicity` 支持。即使在强一致性的模式下，CQL 交易只支持 leader 节点。如果你想要 "读取 `v`, `v++`, 并行写回 `v` ", 仅有的办法是 "从 leader 读取 `v` , `v++`, 从 leader 写回 `v`"

- <a name="fuse">FUSE</a>: CQL 有一个 [保险丝](https://github.com/CovenantSQL/CovenantSQL/tree/develop/cmd/cql-fuse)，从 CockroachDB 获取支持. 这个性能不是很理想，仍然存在一些问题。但它可以通过fio测试，如：

  ```bash
  fio --debug=io --loops=1 --size=8m --filename=../mnt/fiotest.tmp --stonewall --direct=1 --name=Seqread --bs=128k --rw=read --name=Seqwrite --bs=128k --rw=write --name=4krandread --bs=4k --rw=randread --name=4krandwrite --bs=4k --rw=randwrite
  ```


## 样本

- 去中心化论坛：[CovenantForum](https://demo.covenantsql.io/forum/)
- 推特上链：[Twitter Bot @iBlockPin](https://twitter.com/iblockpin)
- 微博上链：[Weibo Bot @BlockPin](https://weibo.com/BlockPin)
- Markdown 同步上链：[Markdown Editor with CovenantSQL sync](https://github.com/CovenantSQL/stackedit)
- 网页端管理：[Web Admin for CovenantSQL](https://github.com/CovenantSQL/adminer)
- CovenantSQL 介绍视频：[How CovenantSQL works(video)](https://youtu.be/2Mz5POxxaQM?t=106)

## 使用案例

### 传统 App

#### 数据隐私

如果你是一个使用密码管理工具的开发者，比如 [1Password](https://1password.com/) or [LastPass](https://www.lastpass.com/). 你可以使用 CQL 作为一个数据库并有以下优点：

1. 无服务器: 不需要部署服务器来存储用户密码，以进行同步，这是一个烫手山芋。
2. 安全: CQL 可以保证所有的加密工作，去中心化的数据存储给予用户更多信心。
3. 合规: CQL 天然符合 [GDPR](https://en.wikipedia.org/wiki/General_Data_Protection_Regulation) 标准。



#### 物联网存储

CQL 矿工全球化部署，物联网节点可以写入最近的矿工。 IoT node can write to nearest CQL miner directly.

1. 实惠: 不需要通过网关传输流量，你可以节省大量的带宽费用。同时，SQL 具有共享经济的特性，以此让存储更实惠。
2. 更快: CQL 共识协议是基于互联网而设计，网络延迟不可避免。


#### 开放数据服务

例如，你是一个非常在意细节的比特币 OHLC 数据维护者，你可以直接放一个 SQL 接口给你的用户以满足各种查询需求。

1. CQL 在平衡数据安全性的同时，可以限制特定的 SQL 查询语句以满足需求。
2. CQL 将 SQL 操作，增删查改等记录在区块链上，方便用户检查账单比如 [这个](https://explorer.dbhub.org/dbs/7a51191ae06afa22595b3904dc558d41057a279393b22650a95a3fc610e1e2df/requests/f466f7bf89d4dd1ece7849ef3cbe5c619c2e6e793c65b31966dbe4c7db0bb072)
3. 对于那些对高性能有要求的客户，可以在用户那里部署 `Slave` 节点，以满足低延迟查询的需求，同时实现几乎实时的数据更新。

#### 安全存储

由于 CQL 数据历史是不可篡改的，CQL 可以用来存储敏感的操作日志，以防止黑客攻击和删除访问日志。


### 去中心化应用（ĐApp）

在比特币或以太坊这种传统公链上存储数据非常昂贵（2018-05-15 以太坊上为4305美元/ MB），而且由于缺乏结构化数据的支持，代码实现非常复杂。但 CQL 可以给你一个结构化且低成本的 SQL 数据库，并为 ĐApp 提供了与真实世界交换数据的空间。


## 文章
团队成员发表过的论文

- [迅雷水晶：一种新颖的基于众筹的内容分发平台](https://dl.acm.org/citation.cfm?id=2736085)
- [基于众筹的视频服务系统性能分析](https://ieeexplore.ieee.org/abstract/document/7114727/)
- [迅雷水晶性能分析：基于众筹的视频分发平台](https://ieeexplore.ieee.org/abstract/document/7762143/)


这些启发了我们：

- [比特币：P2P电子现金系统](https://bitcoin.org/bitcoin.pdf)
- [S/Kademlia](https://github.com/thunderdb/research/wiki/Secure-Kademlia)
    - [S/Kademlia: 一种针对密钥的实用方法](https://ieeexplore.ieee.org/document/4447808/)
- [vSQL: 验证动态外包数据库上的任意SQL查询](https://ieeexplore.ieee.org/abstract/document/7958614/)


## Libs

### 网络栈

[DH-RPC](rpc/) := TLS - Cert + DHT

| 层              | 应用 |
|:-------------------|:--------------:|
| 远程调用协议                |     `net/rpc`    |
| 寻址             |      [**C**onsistent **S**ecure **DHT**](https://godoc.org/github.com/CovenantSQL/CovenantSQL/consistent)     |
| 会话池           |  Session Pool  |
| 多路复用          |      [smux](https://github.com/xtaci/smux)     |
| 传输安全 |      [**E**nhanced **TLS**](https://github.com/CovenantSQL/research/wiki/ETLS(Enhanced-Transport-Layer-Security))      |
| 网络            |       TCP or KCP for optional later      |


#### 测试工具
  - [全球网络拓扑模拟器(GNTE)](https://github.com/CovenantSQL/GNTE) 用于网络模拟
  - [线性一致性测试](https://github.com/anishathalye/porcupine)


#### 接口

CovenantSQL仍在建设中，测试网已经发布，[尝试一下](https://developers.covenantsql.io/docs/quickstart).


- [Golang](client/)
- [Java](https://github.com/CovenantSQL/covenant-connector)
- [NodeJS](https://github.com/CovenantSQL/node-covenantsql)
- [Python](https://github.com/CovenantSQL/python-driver)
- [Microsoft Excel (来自社区)](https://github.com/melancholiaforever/CQL_Excel)
- Coding for more……

关注我们或[![follow on Twitter](https://img.shields.io/twitter/url/https/twitter.com/fold_left.svg?style=social&label=Follow%20%40CovenantLabs)](https://twitter.com/intent/follow?screen_name=CovenantLabs) 保持更新

## 测试网

- [快捷入口](https://developers.covenantsql.io)
- [主链浏览器](http://scan.covenantsql.io)
- [SQL 链浏览器](https://explorer.dbhub.org)
- [Demo & 论坛](https://demo.covenantsql.io/forum/)

## 联系我们

- [博客](https://medium.com/@covenant_labs)
- [YouTube](https://www.youtube.com/channel/UCe9P_TMiexSHW2GGV5qBmZw)
- [邮箱地址](mailto:webmaster@covenantsql.io)
- [论坛](https://demo.covenantsql.io/forum/)
- <a href="https://twitter.com/intent/follow?screen_name=CovenantLabs">
          <img src="https://img.shields.io/twitter/url/https/twitter.com/fold_left.svg?style=social&label=Follow%20%40CovenantLabs"
              alt="follow on Twitter"></a>

- [![Join the chat at https://gitter.im/CovenantSQL/CovenantSQL](https://badges.gitter.im/CovenantSQL/CovenantSQL.svg)](https://gitter.im/CovenantSQL/CovenantSQL?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
