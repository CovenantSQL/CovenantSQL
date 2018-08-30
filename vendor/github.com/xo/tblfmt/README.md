# About

`tblfmt` provides a streaming, bufferred table encoder for result sets (ie,
from a database).

Built mainly for use by [`usql`][usql].

`tblfmt` produces tables like the following:

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

`tblfmt` can be used similar to the following:

```go
// example/main.go
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

	err = tblfmt.NewTableEncoder(result, tblfmt.WithWidths([]int{20, 20})).Encode(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
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
