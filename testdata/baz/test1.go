package baz

import (
	. "context"

	contextx "github.com/roy2220/proxyz/testdata/baz/context"
)

type Test1C struct {
	Err
}

type Err = error

func (*Test1C) C1(int, ...struct{}) (int, func(contextx.Ctxt) func(Context)) { return 0, nil }

func (Test1C) C2(_ []string, haha string) (a, b float64) { return 0, 0 }
