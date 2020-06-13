package bar

import (
	"io"

	. "github.com/roy2220/proxyz/testdata/baz"
)

type Test1B struct {
	Test1C
	io.Reader
}

func (*Test1B) B1() {}

func (Test1B) B2() {}
