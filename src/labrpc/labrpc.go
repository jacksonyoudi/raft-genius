package labrpc

import (
	"bytes"
	"course/labgob"
	"log"
	"reflect"
	"sync"
)

type reqMsg struct {
	endname  interface{}
	svcMeth  string
	argsType reflect.Type
	args     []byte
	replyCh  chan replyMsg
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

func (e *ClientEnd) Call(svcMeth string, args interface{}, reply interface{}) bool {
	req := reqMsg{}
	req.endname = e.endname
	req.svcMeth = svcMeth
	req.argsType = reflect.TypeOf(args)
	req.replyCh = make(chan replyMsg)

	qb := new(bytes.Buffer)
	qe := labgob.NewEncoder(qb)
	if err := qe.Encode(args); err != nil {
		panic(err)
	}
	req.args = qb.Bytes()

	select {
	case e.ch <- req:
	//
	case <-e.done:
		return false
	}
	rep := <-req.replyCh
	if rep.ok {
		rb := bytes.NewBuffer(rep.reply)
		rd := labgob.NewDecoder(rb)
		if err := rd.Decode(reply); err != nil {
			log.Fatalf("ClientEnd.Call(): decode reply: %v\n", err)
		}
		return true
	} else {
		return false
	}

}

type Network struct {
	mu             sync.Mutex
	reliable       bool
	longDelays     bool                        // pause a long time on send on disabled connection
	longReordering bool                        // sometimes delay replies a long time
	ends           map[interface{}]*ClientEnd  // ends, by name
	enabled        map[interface{}]bool        // by end name
	servers        map[interface{}]*Server     // servers, by name
	connections    map[interface{}]interface{} // endname -> servername
	endCh          chan reqMsg
	done           chan struct{} // closed when Network is cleaned up
	count          int32         // total RPC count, for statistics
	bytes          int64         // total bytes send, for statistics

}

type Server struct {
	mu       sync.Mutex
	services map[string]*Service
	count    int // incoming RPCs
}

type Service struct {
	name    string
	rcvr    reflect.Value
	typ     reflect.Type
	methods map[string]reflect.Method
}
