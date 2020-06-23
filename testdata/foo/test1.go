package foo

import (
	"net/http"

	"github.com/roy2220/proxyz/testdata/bar"
)

type Test1A bar.Test1B

func (*Test1A) A1() {}

type TT interface {
	Test(http.File)
}
