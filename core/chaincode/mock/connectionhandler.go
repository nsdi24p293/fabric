// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"sync"

	"github.com/osdi23p228/fabric/core/chaincode/extcc"
	"github.com/osdi23p228/fabric/core/container/ccintf"
)

type ConnectionHandler struct {
	StreamStub        func(string, *ccintf.ChaincodeServerInfo, extcc.StreamHandler) error
	streamMutex       sync.RWMutex
	streamArgsForCall []struct {
		arg1 string
		arg2 *ccintf.ChaincodeServerInfo
		arg3 extcc.StreamHandler
	}
	streamReturns struct {
		result1 error
	}
	streamReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *ConnectionHandler) Stream(arg1 string, arg2 *ccintf.ChaincodeServerInfo, arg3 extcc.StreamHandler) error {
	fake.streamMutex.Lock()
	ret, specificReturn := fake.streamReturnsOnCall[len(fake.streamArgsForCall)]
	fake.streamArgsForCall = append(fake.streamArgsForCall, struct {
		arg1 string
		arg2 *ccintf.ChaincodeServerInfo
		arg3 extcc.StreamHandler
	}{arg1, arg2, arg3})
	fake.recordInvocation("Stream", []interface{}{arg1, arg2, arg3})
	fake.streamMutex.Unlock()
	if fake.StreamStub != nil {
		return fake.StreamStub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.streamReturns
	return fakeReturns.result1
}

func (fake *ConnectionHandler) StreamCallCount() int {
	fake.streamMutex.RLock()
	defer fake.streamMutex.RUnlock()
	return len(fake.streamArgsForCall)
}

func (fake *ConnectionHandler) StreamCalls(stub func(string, *ccintf.ChaincodeServerInfo, extcc.StreamHandler) error) {
	fake.streamMutex.Lock()
	defer fake.streamMutex.Unlock()
	fake.StreamStub = stub
}

func (fake *ConnectionHandler) StreamArgsForCall(i int) (string, *ccintf.ChaincodeServerInfo, extcc.StreamHandler) {
	fake.streamMutex.RLock()
	defer fake.streamMutex.RUnlock()
	argsForCall := fake.streamArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *ConnectionHandler) StreamReturns(result1 error) {
	fake.streamMutex.Lock()
	defer fake.streamMutex.Unlock()
	fake.StreamStub = nil
	fake.streamReturns = struct {
		result1 error
	}{result1}
}

func (fake *ConnectionHandler) StreamReturnsOnCall(i int, result1 error) {
	fake.streamMutex.Lock()
	defer fake.streamMutex.Unlock()
	fake.StreamStub = nil
	if fake.streamReturnsOnCall == nil {
		fake.streamReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.streamReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *ConnectionHandler) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.streamMutex.RLock()
	defer fake.streamMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *ConnectionHandler) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}
