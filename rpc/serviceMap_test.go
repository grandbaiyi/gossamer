package rpc

import (
	"net/http"
	"reflect"
	"testing"
)

// ---------------------- Mock Services -----------------------

type MockServiceA struct {}

type MockServiceAArgs struct {}

type MockServiceAReply struct {
	value string
}
type MockServiceB struct {}

type MockServiceBArgs struct {
	a int
	b int
}
type MockServiceBReply struct {
	value int
}
func (s *MockServiceA) Method1 (req *http.Request, args *MockServiceAArgs, res *MockServiceAReply) error {
	res.value = "method1"
	return nil
}

func TestTypeEvaluators(t *testing.T) {
	ok := isExported("someFunction")
	if ok {
		t.Errorf("function is not exported")
	}

	ok = isExported("SomeFunction")
	if !ok {
		t.Errorf("function is exported")
	}
	// Builtin value
	var i = 10
	typeInt := reflect.TypeOf(i)
	ok = isBuiltin(typeInt)
	if !ok {
		t.Errorf("type %t is builtin", typeInt)
	}
	// Non-builtin pointer
	typePtr := reflect.TypeOf(&MockServiceA{})
	ok = isBuiltin(typePtr)
	if ok {
		t.Errorf("type %t is not a builtin", typePtr)
	}
	// Builtin pointer
	var iPtr = new(int)
	iPtrType := reflect.TypeOf(iPtr)
	ok = isBuiltin(iPtrType)
	if !ok {
		t.Errorf("type %t is a builtin", iPtrType)
	}

}

func TestServiceMap(t *testing.T) {
	s := new(serviceMap)
	mockServiceA := new(MockServiceA)

	err := s.register(mockServiceA, "")
	if err == nil {
		t.Errorf("should not allow empty service names")
	}

	err = s.register(new(MockServiceA), "mockA")
	if err != nil {
		t.Fatalf("could not register: %s", err)
	}

	err = s.register(new(MockServiceB), "mockA")
	if len(err.Error()) <= 1 {
		t.Fatalf("should return an error as there is already a service registered with mockA")
	}

	err = s.register(new(MockServiceB), "mockB")
	if len(err.Error()) <= 1 {
		t.Fatalf("should return an error as there are no methods for this service")
	}

	srvc, srvcMethod, err := s.get("mockA_method1")
	if err != nil {
		t.Fatalf("could not get method %s: %s", "mockA_method1", err)
	}
	_, _, err = s.get("mockA_method1_test")
	if len(err.Error()) <= 1 {
		t.Fatalf("should return an error as thats inproper input")
	}

	_, _, err = s.get("mockA_method2")
	if len(err.Error()) <= 1 {
		t.Fatalf("should return an error as there is no method named method2")
	}

	_, _, err = s.get("mockC_method1")
	if len(err.Error()) <= 1 {
		t.Fatalf("should return an error as there is no service named mockC")
	}

	if srvcMethod.alias != "method1" {
		t.Errorf("incorrect alias. expected: %s got: %s", "method1", srvcMethod.alias)
	}
	reply := reflect.New(reflect.TypeOf(MockServiceAReply{}))
	req := reflect.New(reflect.TypeOf(http.Request{}))
	args := reflect.New(reflect.TypeOf(MockServiceAArgs{}))
	srvcMethod.method.Func.Call([]reflect.Value{srvc.rcvr, req, args, reply})
	srvcReply := reply.Interface().(*MockServiceAReply)

	if srvcReply.value != "method1" {
		t.Errorf("incorrect return value. expected: %s, got: %s", "method1", srvcReply.value)
	}

}