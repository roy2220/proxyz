package main

import (
	"github.com/roy2220/proxyz/testdata/foo"
)

func main() {
	var t foo.Test1A
	t.A1()
	t.A2()
	r1, r2 := t.C1(int(0), struct{}{}, struct{}{})
	r3, r4 := t.C2([]string(nil), "")
	_ = foo.Test1A.Read
	_ = foo.Test1A.Error

	tp := NewTest1AProxy(&t)
	tp.A1()
	tp.A2()
	r1, r2 = tp.C1(int(0), struct{}{}, struct{}{})
	r3, r4 = tp.C2([]string(nil), "")
	_ = tp.Read
	_ = tp.Error
	_, _, _, _ = r1, r2, r3, r4

	if tp.XxxNumberOfMethods() != 6 {
		panic(nil)
	}
}
