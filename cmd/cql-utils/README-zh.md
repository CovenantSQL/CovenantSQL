cql-utils 是 CovenantSQL 的一个命令行工具，具体用法如下。

## 安装
下载 [最新发布版本](https://github.com/CovenantSQL/CovenantSQL/releases) 或直接从源码编译：
```bash
$ go get github.com/CovenantSQL/CovenantSQL/cmd/cql-utils
```
*保证 Golang 环境变量 `$GOPATH/bin` 已在 `$PATH` 中*

## 使用
### 生成公私钥对

```
$ cql-utils -tool confgen
Enter master key(press Enter for default: ""): 
⏎
Private key file: private.key
Public key's hex: 03bc9e90e3301a2f5ae52bfa1f9e033cde81b6b6e7188b11831562bf5847bff4c0
```

生成的 ~/.cql/private.key 文件即是使用主密码加密过的私钥文件，而输出到屏幕上的字符串就是使用十六进制进行编码的公钥。

### 使用私钥文件或公钥生成钱包地址

```
$ cql-utils -tool addrgen
Enter master key(default: ""):
⏎
wallet address: 4jXvNvPHKNPU8Sncz5u5F5WSGcgXmzC1g8RuAXTCJzLsbF9Dsf9
$ cql-utils -tool addrgen -public 02f2707c1c6955a9019cd9d02ade37b931fbfa286a1163dfc1de965ec01a5c4ff8
wallet address: 4jXvNvPHKNPU8Sncz5u5F5WSGcgXmzC1g8RuAXTCJzLsbF9Dsf9
```

你也可以通过-private指定私钥文件，或者把上述的公钥十六进制编码字符串作为命令行参数来直接生成钱包地址。
