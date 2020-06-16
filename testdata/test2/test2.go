package main

import (
	. "github.com/roy2220/proxyz/testdata/foo"
	"net/http"
)

type File = File2
type Server = http.Server
type Err2 = ErrX
type Err3 ErrY
type Err4 ErrZ

func main() {
	p := NewFileWrap(nil)
	_ = p.Read
}
