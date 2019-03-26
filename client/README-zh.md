本文档介绍 CovenantSQL 客户端的使用方式。客户端用来创建、查询、更新和删除 SQLChain 以及绑定的数据库。

## 开始之前

确保 `$GOPATH/bin` 目录在环境变量 `$PATH` 中，执行以下命令

```bash
$ go get github.com/CovenantSQL/CovenantSQL/client
$ go get github.com/CovenantSQL/CovenantSQL/cmd/cql
```

然后在你的 go 代码中 import 第一个 `client` 包。


## 初始化一个 CovenantSQL 客户端

首先需要一个 config 文件和 master key 来初始化。master key 用来加密解密本地密钥对。以下是如何用一个自定义 master key 来生成默认的 config 文件：

### 生成默认的配置文件

运行以下 `cql` 命令，输入 master key（类似密码）来生成本地密钥对。等待几十秒，会在 `~/.cql` 文件夹中，生成一个私钥文件和一个名为 `config.yaml` 的配置文件。

```bash
$ cql generate config
Generating key pair...
Enter master key(press Enter for default: ""):
⏎
Private key file: ~/.cql/private.key
Public key's hex: 025abec9b0072615170f4acf4a2fa1162a13864bb66bc3f140b29f6bf50ceafc75
Generated key pair.
Generating nonce...
INFO[0005] cpu: 1
INFO[0005] position: 0, shift: 0x0, i: 0
nonce: {{1450338416 0 0 0} 26 0000002dd8bdb50ba0270642e4c4bc593c1630ef7784653f311b3c3d6374e514}
node id: 0000002dd8bdb50ba0270642e4c4bc593c1630ef7784653f311b3c3d6374e514
Generated nonce.
Generating config file...
Generated config.
```

有了配置文件之后，可以通过以下 go 代码来初始化 CovenantSQL 客户端：

```go
client.Init(configFile, masterKey)
```

## 客户端使用方式

### 创建一个 SQLChain 数据库

创建 SQLChain 数据库需要指明需要几个节点（nodeCount变量）:

```go
var (
	dsn string
	meta client.ResourceMeta
)
meta.Node = uint16(nodeCount)
dsn, err = client.Create(meta)
// process err
```
创建完毕会返回一个 dsn 字符串，用来访问这个数据库。

### 查询和执行

拿到 dsn 字符串后，可以通过以下代码在 SQLChain 中执行 SQL 语句：

```go

	db, err := sql.Open("covenantsql", dsn)
	// process err

	_, err = db.Exec("CREATE TABLE testSimple ( column int );")
	// process err

	_, err = db.Exec("INSERT INTO testSimple VALUES(?);", 42)
	// process err

	row := db.QueryRow("SELECT column FROM testSimple LIMIT 1;")

	var result int
	err = row.Scan(&result)
	// process err
	fmt.Printf("SELECT column FROM testSimple LIMIT 1; result %d\n", result)

	err = db.Close()
	// process err

```
用法和其他 go sql driver 一致。

### 删除数据库

使用 dsn 来删除数据库：

```go
	err = client.Drop(dsn)
	// process err
```

### 完整示例

在以下目录中有一个简单示例和复杂示例可以参考 [示例](https://github.com/CovenantSQL/CovenantSQL/tree/develop/client/_example)
