package labrpc

import "reflect"

type reqMsg struct {
	endname  interface{}
	svcMeth  string
	argsType reflect.Type
	args     []byte
	replyCh  chan reqMsg
}

type replyMsg struct {
	ok    bool
	reply []byte
}

type ClientEnd struct {
	endname interface{}
	ch      chan reqMsg
	done    chan struct{}
}
