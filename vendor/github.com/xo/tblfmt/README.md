# About [![GoDoc][godoc]][godoc-link] [![Build Status][travis-ci]][travis-ci-link]

[godoc]: https://godoc.org/github.com/xo/tblfmt?status.svg (GoDoc)
[travis-ci]: https://travis-ci.org/xo/tblfmt.svg?branch=master (Travis CI)
[godoc-link]: https://godoc.org/github.com/xo/tblfmt
[travis-ci-link]: https://travis-ci.org/xo/tblfmt

`tblfmt` provides a streaming table encoders for result sets (ie, from a
database), creating tables like the following:

```text
 author_id | name                  | z
-----------+-----------------------+---
        14 | a	b	c	d  |
        15 | aoeu                 +|
           | test                 +|
           |                       |
        16 | foo\bbar              |
        17 | a	b	\r        +|
           | 	a                  |
        18 | 袈	袈		袈 |
        19 | 袈	袈		袈+| a+
           |                       |
(6 rows)
```

## Installing

Install in the usual [Go][go-project] fashion:

```sh
$ go get -u github.com/xo/tblfmt
```

## Using

`tblfmt` was designed for use by [usql][] and Go's native `database/sql` types,
but will handle any type with the following interface:

```go
// ResultSet is the shared interface for a result set.
type ResultSet interface {
	Next() bool
	Scan(...interface{}) error
	Columns() ([]string, error)
	Close() error
	Err() error
	NextResultSet() bool
}
```

`tblfmt` can be used similar to the following:

```go
// _example/main.go
package main

import (
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/xo/dburl"
	"github.com/xo/tblfmt"
)

func main() {
	db, err := dburl.Open("postgres://booktest:booktest@localhost")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	result, err := db.Query("select * from authors")
	if err != nil {
		log.Fatal(err)
	}
	defer result.Close()

	enc, err := tblfmt.NewTableEncoder(result,
		// force minimum column widths
		tblfmt.WithWidths([]int{20, 20}),
	)

	if err = enc.EncodeAll(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
```

Which can produce output like the following:

```text
╔══════════════════════╦═══════════════════════════╦═══╗
║ author_id            ║ name                      ║ z ║
╠══════════════════════╬═══════════════════════════╬═══╣
║                   14 ║ a	b	c	d  ║   ║
║                   15 ║ aoeu                     ↵║   ║
║                      ║ test                     ↵║   ║
║                      ║                           ║   ║
║                    2 ║ 袈	袈		袈 ║   ║
╚══════════════════════╩═══════════════════════════╩═══╝
(3 rows)
```

Please see the [GoDoc listing][godoc] for the full API.

## TODO

1. add center alignment output
2. allow user to override alignment
3. finish template implementations for HTML, LaTeX, etc.
4. add title for tables
5. finish compatibility with PostgreSQL (`psql`) output

[go-project]: https://golang.org/project
[godoc]: https://godoc.org/github.com/xo/tblfmt
[usql]: https://github.com/xo/usql
