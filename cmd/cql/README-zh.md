本文档主要介绍 CovenantSQL 命令行客户端 `cql` 的使用。`cql` 是一个用于批量进行 SQLChain 上数据库的创建、查询、更新或删除操作的命令行工具。

## 安装
下载 [最新发布版本](https://github.com/CovenantSQL/CovenantSQL/releases) 或直接从源码编译：

```bash
$ go get github.com/CovenantSQL/CovenantSQL/client
$ go get github.com/CovenantSQL/CovenantSQL/cmd/cql
```
*保证 Golang 环境变量 `$GOPATH/bin` 已在 `$PATH` 中*

## 生成默认配置文件

首先需要一个 config 文件和由你输入的主密码（master key）来初始化，其中主密码用来加密解密本地密钥对。使用 `cql-utils` 工具进行配置文件生成后，你可以在生成的配置文件目录下找到密钥文件。

具体请参考: [cql-utils 使用文档](https://github.com/CovenantSQL/docs/blob/master/docs/development-cmd-cql-utils-zh.md#使用) 中配置文件及钱包地址生成相关章节。

## 检查钱包余额

使用 `cql` 命令来检查钱包余额：
```bash
$ cql -get-balance
INFO[0000] 
### Public Key ###
0388954cf083bb6bb2b9c7248849b57c76326296fcc0d69764fc61eedb5b8d820c
### Public Key ###
  caller="privatekeystore.go:116 crypto/kms.InitLocalKeyPair"
INFO[0000] stable coin balance is: 100                   caller="main.go:246 main.main"
INFO[0000] covenant coin balance is: 0                   caller="main.go:247 main.main"
```
这里我们得到结果 **"stable coin balance is: 100"**。

## 初始化一个 CovenantSQL 数据库

准备好配置文件和主密码后就可以使用 `cql` 命令来创建数据库了，你的数据库 ID 将会输出到屏幕上：

```bash
# if a non-default password applied on master key, use `-password` to pass it
$ cql -create 1
INFO[0000]
### Public Key ###
039bc931161383c994ab9b81e95ddc1494b0efeb1cb735bb91e1043a1d6b98ebfd
### Public Key ###
  caller="privatekeystore.go:116 crypto/kms.InitLocalKeyPair"
INFO[0000] the newly created database is: covenantsql://0e9103318821b027f35b96c4fd5562683543276b72c488966d616bfe0fe4d213  caller="main.go:297 main.main"
```

这里 `-create 1` 表示创建一个单节点的 SQLChain。

```bash
$ cql -dsn covenantsql://address
```
`address` 就是你的数据库 ID。

`cql` 命令的详细使用帮助如下：

```bash
$ cql -help
```

## 使用 `cql`

现在可以使用 `cql` 进行数据库操作了:

```bash
co:address=> show tables;
```
