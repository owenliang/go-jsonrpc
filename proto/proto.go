package proto

import "sync"

type JsonrpcHandler struct {
	mutex sync.Mutex
	count int
}

type NoArgs struct {}

type NoReturn struct {}

func (s *JsonrpcHandler)Add(args *NoArgs, ret *NoReturn) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count++
	return nil
}

func (s *JsonrpcHandler)Get(args *NoArgs, count *int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	*count = s.count
	return nil
}

func NewJsonrpcHandler() *JsonrpcHandler {
	return &JsonrpcHandler{count: 0}
}