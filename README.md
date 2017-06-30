# verifysfv
a tiny, fast, almost-always-io-bound tool for verifying 
[SFV files](https://en.wikipedia.org/wiki/Simple_file_verification).
Written in [Go](http://golang.org) and adapted from @mpolden's [sfv package](https://github.com/mpolden/sfv)
to verify any of Golang's supported crc32c polynomials (crc32c, IEEE, or Koopman) in parallel.
All deps are vendored, so this should build outside of a $GOPATH for any go>=1.7.

## Installation

`go install github.com/cwlbraa/verifysfv`

## Example

```shell
# to verify an crc32c.sfv file:
verifysfv fileManifest.sfv
# for more options:
verifysfv -h
```
