# sqlparser [![Report card](https://goreportcard.com/badge/github.com/CovenantSQL/sqlparser)](https://goreportcard.com/report/github.com/CovenantSQL/sqlparser) [![GoDoc](https://godoc.org/github.com/CovenantSQL/sqlparser?status.svg)](https://godoc.org/github.com/CovenantSQL/sqlparser)

Go package for parsing SQLite or SQL-92 queries.

## Notice

The backbone of this repo is extracted from [vitessio/vitess](https://github.com/vitessio/vitess).

Inside vitessio/vitess there is a very nicely written sql parser. However as it's not a self-contained application, I created this one.
It applies the same LICENSE as vitessio/vitess.

## Usage

```go
import (
    "github.com/CovenantSQL/sqlparser"
)
```

Then use:

```go
sql := "SELECT * FROM table WHERE a = 'abc'"
stmt, err := sqlparser.Parse(sql)
if err != nil {
	// Do something with the err
}

// Otherwise do something with stmt
switch stmt := stmt.(type) {
case *sqlparser.Select:
	_ = stmt
case *sqlparser.Insert:
}
```

Alternative to read many queries from a io.Reader:

```go
r := strings.NewReader("INSERT INTO table1 VALUES (1, 'a'); INSERT INTO table2 VALUES (3, 4);")

tokens := sqlparser.NewTokenizer(r)
for {
	stmt, err := sqlparser.ParseNext(tokens)
	if err == io.EOF {
		break
	}
	// Do something with stmt or err.
}
```


