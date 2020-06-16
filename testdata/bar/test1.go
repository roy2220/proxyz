package bar

import (
	"io"

	. "github.com/roy2220/proxyz/testdata/baz"
)

type Test1B Test1BReal

type Test1BReal struct {
	Test1C
	io.Reader
	error
}

func (*Test1B) B1() {}

func (Test1B) B2() {}
