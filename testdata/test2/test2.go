package main

import (
	. "github.com/roy2220/proxyz/testdata/foo"
	"net/http"
)

type File = File2
type Server = http.Server

func main() {
	p := NewFileWrap(nil)
	_ = p.Read
}
