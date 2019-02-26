package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"github.com/dato-live/golazy/server/protos"
	"github.com/dato-live/golazy/server/store"
	"github.com/dato-live/golazy/server/store/types"
	"go.uber.org/zap"
	"sync"
	"time"
)

// SessionStore holds live sessions. Long polling sessions are stored in a linked list with
// most recent sessions on top. In addition all sessions are stored in a map indexed by session ID.
type SessionStore struct {
	lock sync.Mutex

	// Support for long polling sessions: a list of sessions sorted by last access time.
	// Needed for cleaning abandoned sessions.
	lru      *list.List
	lifeTime time.Duration

	uidGen types.UidGenerator
	// All sessions indexed by session ID
	sessCache map[string]*Session
}

func (ss *SessionStore) NewSession(conn interface{}, sid string) (*Session, int) {
	var s Session

	s.sid = sid
	if s.sid == "" {
		newSid, err := ss.uidGen.NewSessionUid()
		if err != nil {
			logger.Fatal("SessionStore.NewSession() generate uid failed", zap.Error(err))
		}
		s.sid = newSid
	}

	switch c := conn.(type) {
	case golazy.Node_MessageLoopServer:
		s.proto = GRPC
		s.grpcNode = c
	default:
		s.proto = NONE
	}

	if s.proto != NONE {
		s.send = make(chan interface{}, 1024)
		s.msgSendStatus = make(chan MsgSendStatus, 4096)
		s.stop = make(chan interface{}, 1)
	}

	ss.lock.Lock()
	ss.sessCache[s.sid] = &s
	count := len(ss.sessCache)
	ss.lock.Unlock()
	return &s, count

}

// Get fetches a session from store by session ID.
func (ss *SessionStore) Get(sid string) *Session {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	if sess := ss.sessCache[sid]; sess != nil {
		return sess
	}

	return nil
}

func (ss *SessionStore) GetByClientID(clientID string) *Session {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	for _, v := range ss.sessCache {
		if v.clientInfo.ClientID == clientID {
			return v
		}
	}
	return nil
}

// Delete removes session from store.
func (ss *SessionStore) Delete(s *Session) int {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	delete(ss.sessCache, s.sid)
	return len(ss.sessCache)
}

// Shutdown terminates sessionStore. No need to clean up.
// Don't send to clustered sessions, their servers are not being shut down.
func (ss *SessionStore) Shutdown() {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	for _, s := range ss.sessCache {
		msgID, _ := globals.sessionStore.uidGen.NewMsgUid()
		now := types.TimeNow()
		shutdown := &DMClientMsg{MsgID: msgID, Leave: &DMClientLeave{ClientID: "", Timestamp: &now}}
		if s.stop != nil && s.proto != CLUSTER {
			s.stop <- s.Serialize(shutdown)
		}
	}

	logger.Info(fmt.Sprintf("SessionStore shut down, sessions terminated: %d", len(ss.sessCache)))
}

// NewSessionStore initializes a session store.
func NewSessionStore(lifetime time.Duration) *SessionStore {
	ss := &SessionStore{
		lru:       list.New(),
		lifeTime:  lifetime,
		sessCache: make(map[string]*Session),
	}
	err := ss.uidGen.Init()
	if err != nil {
		logger.Fatal("Init uid generator failed", zap.Error(err))
	}
	return ss
}

//历史失败信息重新发送检查
func RetrySendMsgLoop() {
	for {
		select {
		case <-time.After(time.Second * time.Duration(globals.configs.RetrySecondInterval)):
			reqItems, _ := store.MsgObj.GetRetryReq()
			if reqItems != nil {
				for _, req := range reqItems {
					reqToSess := globals.sessionStore.GetByClientID(req.To)
					if reqToSess != nil {
						var msg DMClientMsg
						err := json.Unmarshal([]byte(req.Content), &msg)
						if err != nil {
							logger.Error("RetrySendMsgLoop Unmarshal Req failed", zap.Error(err))
						} else {
							reqToSess.queueOut(&msg)
						}
						req.Retries += 1
						req.Status = types.StatusRetry

					} else {
						req.Retries += 1
						req.Status = types.StatusFailed
					}

					store.MsgObj.UpdateReq(&req)
				}
			}

			respItems, _ := store.MsgObj.GetRetryResp()
			if respItems != nil {
				for _, resp := range respItems {
					respToSess := globals.sessionStore.GetByClientID(resp.To)
					if respToSess != nil {
						var msg DMClientMsg
						err := json.Unmarshal([]byte(resp.Content), &msg)
						if err != nil {
							logger.Error("RetrySendMsgLoop Unmarshal Resp failed", zap.Error(err))
						} else {
							respToSess.queueOut(&msg)
						}
						resp.Retries += 1
						resp.Status = types.StatusRetry

					} else {
						resp.Retries += 1
						resp.Status = types.StatusFailed
					}

					store.MsgObj.UpdateResp(&resp)
				}
			}
		}
	}
}
