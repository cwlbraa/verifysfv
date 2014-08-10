# gosfv

gosfv is a [Go](http://www.golang.org) library for verifying
[SFV files](https://en.wikipedia.org/wiki/Simple_file_verification).

## Installation

`$ go get github.com/martinp/gosfv`

## Example

```go
package main

import (
	"github.com/martinp/gosfv"
	"log"
)

func main() {
	sfv, err := sfv.Read("/path/to/file.sfv")
	if err != nil {
		log.Fatal(err)
	}

	ok, err := sfv.Verify()
	if err != nil {
		log.Fatal(err)
	}
	if ok {
		log.Print("All files are OK!")
	}
	for _, c := range sfv.Checksums {
		log.Printf("%+v", c)
	}
}
```
