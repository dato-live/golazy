package store

import (
	"errors"
	"github.com/dato-live/golazy/server/adapter"
	"github.com/dato-live/golazy/server/config"
	t "github.com/dato-live/golazy/server/store/types"
	"time"
)

var adp adapter.Adapter
var uGen t.UidGenerator
var configs config.Config

func openAdapter(conf config.Config) error {
	configs = conf
	if adp == nil {
		return errors.New("store: database adapter is missing")
	}

	if adp.IsOpen() {
		return errors.New("store: connection is already opened")
	}

	err := uGen.Init()
	if err != nil {
		return err
	}

	return adp.Open(conf)
}

// Open initializes the persistence system. Adapter holds a connection pool for a database instance.
// 	 name - name of the adapter rquested in the config file
//   jsonconf - configuration string
func Open(conf config.Config) error {
	if err := openAdapter(conf); err != nil {
		return err
	}

	return adp.CheckDbVersion()
}

// Close terminates connection to persistent storage.
func Close() error {
	if adp.IsOpen() {
		return adp.Close()
	}

	return nil
}

// IsOpen checks if persistent storage connection has been initialized.
func IsOpen() bool {
	if adp != nil {
		return adp.IsOpen()
	}

	return false
}

// GetAdapterName returns the name of the current adater.
func GetAdapterName() string {
	if adp != nil {
		return adp.GetName()
	}

	return ""
}

// InitDb creates a new database instance. If 'reset' is true it will first attempt to drop
// existing database. If jsconf is nil it will assume that the connection is already open.
// If it's non-nil, it will use the config string to open the DB connection first.
func InitDb(conf config.Config, reset bool) error {
	if !IsOpen() {
		if err := openAdapter(conf); err != nil {
			return err
		}
	}
	return adp.CreateDb(reset)
}

// RegisterAdapter makes a persistence adapter available.
// If Register is called twice or if the adapter is nil, it panics.
func RegisterAdapter(name string, a adapter.Adapter) {
	if a == nil {
		panic("store: Register adapter is nil")
	}

	if adp != nil {
		panic("store: adapter '" + adp.GetName() + "' is already registered")
	}

	adp = a
}

type MsgObjMapper struct {
}

var MsgObj MsgObjMapper

func (MsgObjMapper) InsertReq(received *t.ReqReceived) error {
	return adp.InsertReq(received)
}

func (MsgObjMapper) InsertResp(received *t.RespReceived) error {
	return adp.InsertResp(received)
}

func (MsgObjMapper) GetReqByMsgID(msgId string) (*t.ReqReceived, error) {
	return adp.GetReqByMsgID(msgId)
}

func (MsgObjMapper) GetRespByMsgID(msgId string) (*t.RespReceived, error) {
	return adp.GetRespByMsgID(msgId)
}

func (MsgObjMapper) UpdateReq(req *t.ReqReceived) error {
	return adp.UpdateReq(req)
}

func (MsgObjMapper) UpdateResp(resp *t.RespReceived) error {
	return adp.UpdateResp(resp)
}

func (MsgObjMapper) DeleteReq(id int64) error {
	return adp.DeleteReq(id)
}

func (MsgObjMapper) DeleteResp(id int64) error {
	return adp.DeleteResp(id)
}

func (MsgObjMapper) DeleteSendedOrExpireMsg() error {
	return adp.DeleteSendedOrExpireMsg()
}

func (MsgObjMapper) GetRetryReq() ([]t.ReqReceived, error) {
	return adp.GetRetryReq()
}

func (MsgObjMapper) GetRetryResp() ([]t.RespReceived, error) {
	return adp.GetRetryResp()
}

func DbClearLoop() {
	for {
		select {
		case <-time.After(time.Minute * time.Duration(configs.CleanDbMinuteInterval)):
			MsgObj.DeleteSendedOrExpireMsg()
		}
	}
}
