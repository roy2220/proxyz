package baz

import . "context"

type Test1C struct {
	error
}

func (*Test1C) C1(int, ...struct{}) (int, func() func(Context)) { return 0, nil }

func (Test1C) C2(_ []string, haha string) (a, b float64) { return 0, 0 }
