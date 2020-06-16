// Package proxyz defines common interfaces and utility for generated code.
package proxyz

// MethodCallInterceptor is the type of function intercepting calls to methods.
type MethodCallInterceptor func(methodCall MethodCall)

// MethodCall represents a call to a method.
type MethodCall interface {
	// Forward forwards the method call to the next interceptor, if any,
	// or forwards to the underlying object to call the real method.
	//
	// It should be manually called in the body of every intercepting
	// function, otherwise the method will NOT be actually called.
	Forward()

	// GetArg returns the argument of the method call at the given index.
	GetArg(argIndex int) (arg interface{})

	// SetArg sets the argument of the method call at the given index.
	SetArg(argIndex int, arg interface{})

	// GetResult returns the result of the method call at the given index.
	GetResult(resultIndex int) (result interface{})

	// SetResult sets the result of the method call at the given index.
	SetResult(resultIndex int, result interface{})

	// MethodName returns the name of the method called.
	MethodName() string

	// MethodIndex returns the index of the method called.
	MethodIndex() int

	// NumberOfArgs returns the number of the arguments of the method call.
	NumberOfArgs() int

	// NumberOfResults returns the number of the results of the method call.
	NumberOfResults() int
}

// Proxy represents a proxy generated.
type Proxy interface {
	// XxxInterceptMethodCall adds an interceptor to intercept the calls
	// to the method at the given index.
	XxxInterceptMethodCall(methodIndex int, methodCallInterceptor MethodCallInterceptor)

	// XxxGetMethodName returns the name of the method at the given index.
	XxxGetMethodName(methodIndex int) string

	// XxxNumberOfMethods returns the number of the methods of the proxy
	// generated, excluding the methods whose names start with 'Xxx'.
	XxxNumberOfMethods() int

	// XxxUnderlyingType returns the representation of the underlying type of
	// the proxy generated.
	XxxUnderlyingType() string
}

// XxxProxyBase represents the base of proxies generated.
type XxxProxyBase struct {
	xxxMethodCallInterceptors methodCallInterceptors
}

// XxxInterceptMethodCall implements Proxy.XxxInterceptMethodCall.
func (pb *XxxProxyBase) XxxInterceptMethodCall(methodIndex int, methodCallInterceptor MethodCallInterceptor) {
	pb.xxxMethodCallInterceptors.AddItem(methodIndex, methodCallInterceptor)
}

// XxxGetMethodCallInterceptors returns the interceptors applied to the calls
// to the method at the given index. It serves for generated code.
func (pb *XxxProxyBase) XxxGetMethodCallInterceptors(methodIndex int) []MethodCallInterceptor {
	return pb.xxxMethodCallInterceptors.GetItems(methodIndex)
}

type methodCallInterceptors struct {
	methodIndex2Items map[int][]MethodCallInterceptor
}

func (mci *methodCallInterceptors) AddItem(methodIndex int, item MethodCallInterceptor) {
	if mci.methodIndex2Items == nil {
		mci.methodIndex2Items = map[int][]MethodCallInterceptor{
			methodIndex: {item},
		}
	} else {
		items := mci.methodIndex2Items[methodIndex]
		mci.methodIndex2Items[methodIndex] = append(items, item)
	}

}

func (mci *methodCallInterceptors) GetItems(methodIndex int) []MethodCallInterceptor {
	return mci.methodIndex2Items[methodIndex]
}
