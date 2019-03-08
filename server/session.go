package main

import (
	"encoding/json"
	"github.com/dato-live/golazy/server/protos"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"log"
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

	// Websocket. Set only for websocket sessions
	ws *websocket.Conn

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

func (s *Session) cleanUp(expired bool) {
	if !expired {
		globals.sessionStore.Delete(s)
	}
	//globals.cluster.sessionGone(s)

}

// Message received, convert bytes to ClientComMessage and dispatch
func (s *Session) dispatchRaw(raw []byte) {
	var msg DMClientMsg

	if len(raw) == 1 && raw[0] == 0x31 {
		// 0x31 == '1'. This is a network probe message. Respond with a '0':
		s.queueOutBytes([]byte{0x30})
		return
	}

	toLog := raw
	truncated := ""
	if len(raw) > 512 {
		toLog = raw[:512]
		truncated = "<...>"
	}
	log.Printf("in: '%s%s' ip='%s' sid='%s' uid='%s'", toLog, truncated, s.remoteAddr, s.sid, s.uid)

	if err := json.Unmarshal(raw, &msg); err != nil {
		// Malformed message
		log.Println("s.dispatch", err, s.sid)
		s.queueOut(ErrMalformed("", "", time.Now().UTC().Round(time.Millisecond)))
		return
	}

	s.dispatch(&msg)
}

func (s *Session) Serialize(msg *DMClientMsg) interface{} {
	if s.proto == GRPC {
		return PbSerialize(msg)
	}
	out, _ := json.Marshal(msg)
	return out
}

// queueOutBytes attempts to send a ServerComMessage already serialized to []byte.
// If the send buffer is full, timeout is 50 usec
func (s *Session) queueOutBytes(data []byte) bool {
	if s == nil {
		return true
	}

	select {
	case s.send <- data:
	case <-time.After(time.Microsecond * 50):
		log.Println("s.queueOutBytes: timeout", s.sid)
		return false
	}
	return true
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
