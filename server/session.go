package main

import (
	"encoding/json"
	"github.com/dato-live/golazy/server/protos"
	"go.uber.org/zap"
	"sync"
	"time"
)

//网络传输类型
const (
	NONE = iota
	WEBSOCK
	LPOLL
	GRPC
	CLUSTER
)

type ClientInfo struct {
	ClientID          string
	ClientName        string
	ClientVersion     string
	ClientDescription string
	AllowedCommandIDs map[int64]string
}

type MsgSendStatus struct {
	MsgType string
	MsgID   string
	IsOk    bool
}

type Session struct {
	// protocol - NONE (unset), WEBSOCK, LPOLL, CLUSTER, GRPC
	proto int

	// gRPC handle. Set only for gRPC clients
	grpcNode golazy.Node_MessageLoopServer

	// IP address of the client. For long polling this is the IP of the last poll
	remoteAddr string

	// Time when the session received any packer from client
	lastAction time.Time

	clientInfo ClientInfo

	// Outbound mesages, buffered.
	// The content must be serialized in format suitable for the session.
	send chan interface{}

	msgSendStatus chan MsgSendStatus

	// Channel for shutting down the session, buffer 1.
	// Content in the same format as for 'send'
	stop chan interface{}

	// Session ID
	sid string

	// Needed for long polling and grpc.
	lock sync.Mutex
}

func (s *Session) Serialize(msg *DMClientMsg) interface{} {
	if s.proto == GRPC {
		return PbSerialize(msg)
	}
	out, _ := json.Marshal(msg)
	return out
}

func (s *Session) queueOut(msg *DMClientMsg) bool {
	if s == nil {
		return true
	}

	select {
	case s.send <- s.Serialize(msg):
	case <-time.After(time.Microsecond * 50):
		logger.Warn("s.queueOut: timeout", zap.String("session", s.sid))
		return false
	}
	return true
}
