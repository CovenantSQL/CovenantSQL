This doc introduce the usage of CovenantSQL adapter. This adapter lets you use CovenantSQL on any platform from any programming languages using http(s) protocol. The CovenantSQL  Java/Python/NodeJS Driver currently is based on adapter to service.

## Prerequisites

Make sure the ```$GOPATH/bin``` is in your ```$PATH```, download build the adapter binary.

```shell
$ go get github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter
```

## Adapter Usage

Adapter can use the same ```config.yaml``` and key pair with `cql`

### Generating Default Config File

First, generate the main configuration file. Same as [Generating Default Config File in Golang Client Doc](https://github.com/CovenantSQL/CovenantSQL/tree/develop/client#generating-default-config-file). An existing configuration file can also be used.

### Start

Start the adapter by following commands:

```shell
$ cql-adapter -listen 127.0.0.1:4661
```

### API

#### Access API

##### Query

###### Read only query

**POST** /v1/query

###### Parameters

**query:** database query

**database:** database id

###### Response

```json
{
    "data": {
        "rows": [
            [
                1,
                1,
            ]
        ]
    },
    "status": "ok",
    "success": true
}
```

### Configure HTTPS Adapter

Adapter use tls certificate for client authorization, a public or self-signed ssl certificate is required for adapter server to start. The adapter config is placed as a ```Adapter``` section of the main config file including following configurable fields.

| Name              | Type     | Description                                                  | Default |
| ----------------- | -------- | ------------------------------------------------------------ | ------- |
| ListenAddr        | string   | adapter server listen address                                |         |
| CertificatePath   | string   | adapter server tls certificate file<br />** all following file path is related to working root |         |
| PrivateKeyPath    | string   | adapter server tls private key file                          |         |
| VerifyCertificate | bool     | should adapter server verify client certificate or not<br />a client custom CA is required, all valid clients certificate should be issued by this CA | false   |
| AdminCerts        | []string | each item requires to be a certificate file path<br />client with configured certificate will be granted with ADMIN privilege<br />ADMIN privilege is able to CREATE/DROP database, send WRITE/READ request |         |
| WriteCerts        | []string | same format as ```AdminCerts ``` field<br />client with configured certificate will be granted with WRITE privilege<br />WRITE privilege is able to send WRITE/READ request only |         |
| StorageDriver     | string   | two available storage driver: ```sqlite3``` and ```covenantsql```, use ```sqlite3``` driver for test purpose only |         |
| StorageRoot       | string   | required by ```sqlite3``` storage driver, database files is placed under this root path, this path is treated as relative to working root |         |

[mkcert](https://github.com/FiloSottile/mkcert) is a handy command to generate tls certificates, run the following command to generate the server certificate.

``````
$ CAROOT=$(pwd) mkcert server
Using the local CA at "/demo" ✨
Warning: the local CA is not installed in the system trust store! ⚠️
Warning: the local CA is not installed in the Firefox trust store! ⚠️
Run "mkcert -install" to avoid verification errors ‼️

Created a new certificate valid for the following names 📜
 - "server"

The certificate is at "./server.pem" and the key at "./server-key.pem" ✅

And move them to ~/.cql/ dir.
``````

You can use following interactive command to generate adapter config.


###### Example

```bash
// send query (linux)
curl -v https://127.0.0.1:4661/v1/query --insecure \
	--cert read.data.covenantsql.io.pem \
	--key read.data.covenantsql.io.key \
	--form database=kucoin.GO.BTC \
	--form query='select * from trades limit 10'

// send query (mac curl build with SecureTransport)
curl -v https://e.morenodes.com:11108/v1/query --insecure \
	--cert 'read.p12:1' \
	--form database=kucoin.GO.BTC \
	--form query='select * from trades limit 10'
	
// response got
{
    "data": {
        "rows": [
            [
                "06e38e29c05bbedaa672c7dc333d9605",
                "2018-07-13T07:16:13Z",
                "buy",
                "1.141e-05",
                "6626.6761"
            ],
            [
                "b0258eb3b0d2743223d536943f964c98",
                "2018-07-13T07:16:29Z",
                "buy",
                "1.141e-05",
                "5270.7347"
            ],
            [
                "c199e34b6d30661225ad602cfc758d18",
                "2018-07-13T07:16:41Z",
                "buy",
                "1.141e-05",
                "5525.498"
            ],
            [
                "83e782aac5a3fc9275dbce8e6142e708",
                "2018-07-13T07:16:48Z",
                "buy",
                "1.146e-05",
                "8367.071"
            ],
            [
                "efbed772382067d6d2ea8c8923e382a9",
                "2018-07-13T07:16:51Z",
                "buy",
                "1.146e-05",
                "4632.929"
            ],
            [
                "160e86aadc5e580ced8c041ac9e0e3c5",
                "2018-07-13T07:16:52Z",
                "buy",
                "1.15e-05",
                "270.9786"
            ],
            [
                "086b19c133a52cf72e05bc33314cc5d4",
                "2018-07-13T07:16:52Z",
                "buy",
                "1.15e-05",
                "15792.9023"
            ],
            [
                "eca962344164c0652148dbff47904083",
                "2018-07-13T07:16:54Z",
                "buy",
                "1.15e-05",
                "6943.1195"
            ],
            [
                "b632795b2bbe7571918e7b175531fc4c",
                "2018-07-13T07:16:59Z",
                "sell",
                "1.15e-05",
                "485.2765"
            ],
            [
                "7ab312cd93ebe8cdf6d9f46997a81271",
                "2018-07-13T07:16:59Z",
                "sell",
                "1.146e-05",
                "4774.1287"
            ]
        ]
    },
    "status": "ok",
    "success": true
}
```

##### Exec

###### Write query

**POST** /v1/exec

###### Parameters

**query:** database query

**database:** database id

###### Response

```json
{
    "data": null,
    "status": "ok",
    "success": true
}
```

#### Admin API

##### CreateDatabase

###### Create database

**POST** /v1/admin/create

###### Parameters

**node:** node count requirement for CovenantSQL

###### Response

```json
{
    "data": {
        "database": "5345c6cbdee4462a708d51194ff5802d52b3772d28f15bb3215aac76051ec46d"   
    },
    "status": "ok",
    "success": true
}
```

##### DropDatabase

###### Drop database

**DELETE** /v1/admin/drop

###### Parameters

**database:** database id
