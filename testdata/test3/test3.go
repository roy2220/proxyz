package main

import (
	"fmt"
	"strings"

	"github.com/roy2220/proxyz"
)

type Calc struct {
}

func (c *Calc) Sum(a, b int) (s string) {
	s = fmt.Sprintf("%d + %d = %d", a, b, a+b)
	return s
}

func main() {
	var c Calc
	cp := NewCalcProxy(&c)

	cp.XxxInterceptMethodCall(CalcProxySum, func(mc proxyz.MethodCall) {
		a := mc.GetArg(0).(int)
		a += 100
		mc.SetArg(0, a)

		mc.Forward()

	})

	cp.XxxInterceptMethodCall(CalcProxySum, func(mc proxyz.MethodCall) {
		mc.Forward()

		s := mc.GetResult(0).(string)
		s = strings.ReplaceAll(s, "=", "!=")
		mc.SetResult(0, s)
	})

	s := cp.Sum(1, 2)
	if s != "101 + 2 != 103" {
		panic("")
	}
}
