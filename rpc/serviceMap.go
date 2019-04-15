package rpc

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type service struct {
	name	string
	rcvr 	reflect.Value
	rcvrType	reflect.Type
	methods	map[string] *serviceMethod
}

type serviceMethod struct {
	method	reflect.Method
	alias	string
	// TODO: I think these can be removed if we only use 1 args/reply type
	argsType	reflect.Type
	replyType	reflect.Type
}

type serviceMap struct {
	mutex    sync.Mutex
	services map[string]*service
}

// Precompute these for efficiency
var typeOfError   = reflect.TypeOf((*error)(nil)).Elem()
var typeOfRequest = reflect.TypeOf((*http.Request)(nil)).Elem()


func (m *serviceMap) register(rcvr interface{}, name string) error {
	if name == "" {
		return fmt.Errorf("unspecified name, please specify a name")
	}

	if m.services[name] != nil {
		return fmt.Errorf("name is already registered")
	}

	s := &service{
		name: name,
		rcvr: reflect.ValueOf(rcvr),
		rcvrType: reflect.TypeOf(rcvr),
		methods: make(map[string]*serviceMethod),
	}

	for i := 0; i < s.rcvrType.NumMethod(); i++ {
		method := s.rcvrType.Method(i)
		methodType := method.Type

		if method.PkgPath != "" {
			continue
		}
		// Must have receiver and 3 inputs
		if methodType.NumIn() != 4 {
			continue
		}
		// First arg must be http.Request
		reqType := methodType.In(1)
		if reqType.Kind() != reflect.Ptr || reqType.Elem() != typeOfRequest {
			continue
		}
		// Second arg must be pointer and exported
		argsType := methodType.In(2)
		if argsType.Kind() != reflect.Ptr || !isExported(method.Name) || !isBuiltin(argsType) {
			continue
		}
		// Third arg must be pointer and exported
		replyType := methodType.In(3)
		if replyType.Kind() != reflect.Ptr || !isExported(method.Name) || !isBuiltin(replyType) {
			continue
		}
		// Must have 1 return value of type error
		if methodType.NumOut() != 1 {
			continue
		}
		returnType := methodType.Out(0)
		if returnType!= typeOfError {
			continue
		}
		// Force first character to lower case
		methodRunes := []rune(method.Name)
		methodAlias := string(unicode.ToLower(methodRunes[0])) + string(methodRunes[1:])
		m.mutex.Lock()
		s.methods[methodAlias] = &serviceMethod{
			method: method,
			alias: methodAlias,
			argsType: argsType.Elem(),
			replyType: returnType.Elem(),
		}
		m.mutex.Unlock()
	}

	if len(s.methods) == 0 {
		return fmt.Errorf("rpc: expected some methods to register for service %s, got none", s.name)
	}

	if m.services == nil {
		m.services = make(map[string]*service)
	}

	m.services[s.name] = s
	return nil
}

func (m *serviceMap) get(id string) (*service, *serviceMethod, error){
	tokens := strings.Split(id, "_")
	if len(tokens) != 2 {
		err := fmt.Errorf("invalid method name: %s", id)
		return nil, nil, err
	}
	m.mutex.Lock()
	service := m.services[tokens[0]]
	m.mutex.Unlock()
	if service == nil {
		err := fmt.Errorf("service %s not recognized", tokens[0])
		return nil, nil, err
	}
	method := service.methods[tokens[1]]
	if method == nil {
		err := fmt.Errorf("method %s in service %s not recognized", tokens[1], tokens[0])
		return nil, nil, err
	}
	return service, method, nil
}

func isExported(name string) bool {
	firstRune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(firstRune)
}


// TODO: This dont work
func isBuiltin(t reflect.Type) bool {
	// Follow indirection
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// TODO: confirm this works
	// Should be empty if builtin
	return t.PkgPath() == ""
}