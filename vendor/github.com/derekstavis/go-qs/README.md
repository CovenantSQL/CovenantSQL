<img alt="Guitar String" src="https://cloud.githubusercontent.com/assets/1611639/20510991/96515948-b05b-11e6-8eaf-debc9a84f61c.png" align="right" />

# go-qs

Go port of Rack's query strings

[![Build Status](https://travis-ci.org/derekstavis/go-qs.svg?branch=master)](https://travis-ci.org/derekstavis/go-qs)


This package was written as I haven't found a good package that understands
[Rack/Rails](http://guides.rubyonrails.org/form_helpers.html#understanding-parameter-naming-conventions) query string [format](https://gist.github.com/dapplebeforedawn/3724090).

It have been designed to marshal and unmarshal nested query strings from/into
`map[string]interface{}`, inspired on the interface of Go builtin `json`
package.

## Compatibility

`go-qs` is a port of [Rack's code](https://github.com/rack/rack/blob/rack-1.3/lib/rack/utils.rb#L114).
All tests included into [test suite](https://github.com/derekstavis/go-qs/blob/master/marshal_test.go)
are also a port of [Rack tests](https://github.com/rack/rack/blob/rack-1.3/test/spec_utils.rb#L107),
so this package keeps great compatibility with Rack implementation.

## Usage

### Unmarshal

To unmarshal query strings to a `map[string]interface{}`:

```go
package main

import (
  "fmt"
  "github.com/derekstavis/go-qs"
)

query, err := qs.Unmarshal("foo=bar&names[]=foo&names[]=bar")

if err != nil {
  fmt.Printf("%#+v\n", query)
}
```

The output:

```
map[string]interface {}{"foo":"bar", "names":[]interface {}{"foo", "bar"}}
```

### Marshal

You can also marshal a `map[string]interface{}` to a query string:

```go
package main

import (
  "fmt"
  "github.com/derekstavis/go-qs"
)

payload := map[string]interface {}{"foo":"bar", "names":[]interface {}{"foo", "bar"}}

querystring, err := qs.Marshal(payload)

if err != nil {
  fmt.Printf(querystring)
}
```

The output:

```
foo=bar&names[]=foo&names[]=bar
```

## License

```
MIT Copyright (c) 2016 Derek W. Stavis
```
