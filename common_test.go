package proxyz_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/roy2220/proxyz"
)

func TestProxyBase(t *testing.T) {
	pb := proxyz.XxxProxyBase{}
	s := ""
	pb.XxxInterceptMethodCall(101, func(_ proxyz.MethodCall) {
		s += "a"
	})
	pb.XxxInterceptMethodCall(101, func(_ proxyz.MethodCall) {
		s += "b"
	})
	pb.XxxInterceptMethodCall(101, func(_ proxyz.MethodCall) {
		s += "c"
	})
	for _, mci := range pb.XxxGetMethodCallInterceptors(100) {
		mci(nil)
	}
	for _, mci := range pb.XxxGetMethodCallInterceptors(101) {
		mci(nil)
	}
	assert.Equal(t, "abc", s)
}
