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


CovenantSQL是一个基于区块链技术的去中心化众筹式SQL数据库，并具备以下特点：

- **SQL**:大多支持SQL-92
- **去中心化**:使用我们的共识算法DH-RPC和Kayak去中心化
- **隐私**:通过加密和授权许可进行访问
- **不可篡改**:CovenantSQL中的查询历史记录是不可变但是可跟踪的

我们相信[在下一个互联网时代，每个人都应该有完整的**数据权利**](https://medium.com/@covenant_labs/covenantsql-the-sql-database-on-blockchain-db027aaf1e0e)

#### 一行代码接入区块链数据
```go
sql.Open("CovenantSQL", dbURI)
```

## 

![CovenantSQL 3 Layer design](logo/arch.png)

- 第一层: **全局共识层**(主链，架构图中的中间环):
    - 整个网络中只有一个主链。
    - 主要负责数据库矿工与用户的合同匹配，交易结算，反作弊，子链哈希锁定等全局共识事宜。
- 第二层: **SQL共识层**(子链，架构图中的两边环):
    - 每个数据库都有自己独立的子链。
    - 主要负责数据库各种事务的签名，交付和一致性。这里主要实现永久可追溯性的数据历史，并且在主链中执行哈希锁定。
- 第三层: **数据储存层**(支持SQL-92的数据库引擎):
    - 每个数据库都有自己独立的分布式引擎。
    - 主要负责：数据库存储和加密；查询处理和签名；高效索引。

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

CovenantSQL仍在建设中，测试网已经发布，[尝试一下](https://testnet.covenantsql.io/).


- [Golang](client/)
- [Java](https://github.com/CovenantSQL/covenant-connector)
- [NodeJS](https://github.com/CovenantSQL/node-covenantsql)
- [Python](https://github.com/CovenantSQL/python-driver)
- Coding for more……

关注我们或[![follow on Twitter](https://img.shields.io/twitter/url/https/twitter.com/fold_left.svg?style=social&label=Follow%20%40CovenantLabs)](https://twitter.com/intent/follow?screen_name=CovenantLabs) 保持更新

## 测试网

- [快捷入口](https://testnet.covenantsql.io/quickstart)
- [测试网水龙头](https://testnet.covenantsql.io/)

## 联系我们

- [邮箱地址](mailto:webmaster@covenantsql.io)
- <a href="https://twitter.com/intent/follow?screen_name=CovenantLabs">
          <img src="https://img.shields.io/twitter/url/https/twitter.com/fold_left.svg?style=social&label=Follow%20%40CovenantLabs"
              alt="follow on Twitter"></a>

- [![Join the chat at https://gitter.im/CovenantSQL/CovenantSQL](https://badges.gitter.im/CovenantSQL/CovenantSQL.svg)](https://gitter.im/CovenantSQL/CovenantSQL?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
