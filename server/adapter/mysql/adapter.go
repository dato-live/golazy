// +build mysql

package mysql

import (
	"database/sql"
	"errors"
	"github.com/dato-live/golazy/server/config"
	"github.com/dato-live/golazy/server/logs"
	"github.com/dato-live/golazy/server/store"
	t "github.com/dato-live/golazy/server/store/types"
	ms "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"go.uber.org/zap"
)

// adapter保存MySQL连接数据
type adapter struct {
	db      *xorm.Engine
	dsn     string
	dbName  string
	version string
}

const (
	defaultDSN      = "root@tcp(localhost)/golazy?parseTime=true&collation=utf8mb4_unicode_ci"
	defaultDatabase = "golazy"
	dbVersion       = "100"
	adapterName     = "mysql"
)

const (
	maxResults = 1024
)

var logger *zap.Logger
var configs config.Config

func (a *adapter) Open(conf config.Config) error {
	configs = conf
	logger = logs.GetLogger()
	if a.db != nil {
		return errors.New("mysql adapter is already connected")
	}
	var err error
	a.dsn = conf.Store.Adapters.Mysql.DSN
	if a.dsn == "" {
		a.dsn = defaultDSN
	}

	a.dbName = conf.Store.Adapters.Mysql.Database
	if a.dbName == "" {
		a.dbName = defaultDatabase
	}

	a.db, err = xorm.NewEngine("mysql", a.dsn)
	//打印执行的SQL，仅调试模式下输出
	a.db.ShowSQL(configs.ShowSqlToConsole)

	if err != nil {
		return err
	}
	err = a.db.Ping()
	if isMissingDb(err) {
		//忽略数据库找不到的错误，当数据库正在初始化时，找不到数据库是正常的
		err = nil
	}
	return err

}

// Close closes the underlying database connection
func (a *adapter) Close() error {
	var err error
	if a.db != nil {
		err = a.db.Close()
		a.db = nil
		a.version = ""
	}
	return err
}

// IsOpen returns true if connection to database has been established. It does not check if
// connection is actually live.
func (a *adapter) IsOpen() bool {
	return a.db != nil
}

// Read current database version
func (a *adapter) getDbVersion() (string, error) {
	var vers t.KvMeta
	_, err := a.db.Where("key_name = ?", "version").Get(&vers)
	if err != nil {
		if isMissingDb(err) || err == sql.ErrNoRows {
			err = errors.New("Database not initialized")
		}
		return "", err
	}
	a.version = vers.KeyValue

	return a.version, nil
}

// CheckDbVersion checks whether the actual DB version matches the expected version of this adapter.
func (a *adapter) CheckDbVersion() error {
	if a.version == "" {
		_, err := a.getDbVersion()
		if err != nil {
			return err
		}
	}

	if a.version != dbVersion {
		return errors.New("Invalid database version " + a.version +
			". Expected " + dbVersion)
	}

	return nil
}

// GetName returns string that adapter uses to register itself with store.
func (a *adapter) GetName() string {
	return adapterName
}

// CreateDb initializes the storage.
func (a *adapter) CreateDb(reset bool) error {
	err := a.db.Sync2(new(t.KvMeta), new(t.ReqReceived), new(t.RespReceived))
	if err != nil {
		return err
	}
	_, err = a.db.Insert(&t.KvMeta{KeyName: "version", KeyValue: dbVersion})
	return err

}

func (a *adapter) InsertReq(received *t.ReqReceived) error {
	_, err := a.db.Insert(received)
	return err
}

func (a *adapter) InsertResp(received *t.RespReceived) error {
	_, err := a.db.Insert(received)
	return err
}

func (a *adapter) GetReqByMsgID(msgId string) (*t.ReqReceived, error) {
	req := &t.ReqReceived{MsgID: msgId}
	has, err := a.db.Get(req)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("Record not found!")
	}
	return req, nil
}

func (a *adapter) GetRespByMsgID(msgId string) (*t.RespReceived, error) {
	resp := &t.RespReceived{MsgID: msgId}
	has, err := a.db.Get(resp)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errors.New("Record not found!")
	}
	return resp, nil
}

func (a *adapter) UpdateReq(req *t.ReqReceived) error {
	_, err := a.db.Id(req.Id).Update(req)
	return err
}

func (a *adapter) UpdateResp(resp *t.RespReceived) error {
	_, err := a.db.Id(resp.Id).Update(resp)
	return err
}

func (a *adapter) DeleteReq(id int64) error {
	_, err := a.db.Delete(&t.RespReceived{Id: id})
	return err
}

func (a *adapter) DeleteResp(id int64) error {
	_, err := a.db.Delete(&t.RespReceived{Id: id})
	return err
}

func (a *adapter) DeleteSendedOrExpireMsg() error {
	_, err := a.db.Delete(&t.ReqReceived{Status: t.StatusSucceeded})
	if err != nil {
		logger.Error("DeleteSendedOrExpireMsg Delete Req failed", zap.Error(err))
	}
	_, err = a.db.Delete(&t.RespReceived{Status: t.StatusSucceeded})
	if err != nil {
		logger.Error("DeleteSendedOrExpireMsg Delete Resp failed", zap.Error(err))
	}
	_, err = a.db.Where("expires_at>=?", t.TimeNow()).Delete(&t.ReqReceived{})
	if err != nil {
		logger.Error("DeleteSendedOrExpireMsg Delete Req Expire failed", zap.Error(err))
	}
	_, err = a.db.Where("expires_at>=?", t.TimeNow()).Delete(&t.RespReceived{})
	if err != nil {
		logger.Error("DeleteSendedOrExpireMsg Delete Resp Expire failed", zap.Error(err))
	}
	return err
}

func (a *adapter) GetRetryReq() ([]t.ReqReceived, error) {
	items := make([]t.ReqReceived, 0)
	err := a.db.Where("status = ?", t.StatusFailed).And("retries <= ?", configs.MaxRetryCount).Find(&items)
	if err != nil {
		logger.Error("GetRetryReq failed", zap.Error(err))
		return nil, err
	}
	return items, err
}

func (a *adapter) GetRetryResp() ([]t.RespReceived, error) {
	items := make([]t.RespReceived, 0)
	err := a.db.Where("status = ?", t.StatusFailed).And("retries <= ?", configs.MaxRetryCount).Find(&items)
	if err != nil {
		logger.Error("GetRetryResp failed", zap.Error(err))
		return nil, err
	}
	return items, err
}

// Check if MySQL error is a Error Code: 1062. Duplicate entry ... for key ...
func isDupe(err error) bool {
	if err == nil {
		return false
	}

	myerr, ok := err.(*ms.MySQLError)
	return ok && myerr.Number == 1062
}

func isMissingDb(err error) bool {
	if err == nil {
		return false
	}

	myerr, ok := err.(*ms.MySQLError)
	return ok && myerr.Number == 1049
}

func init() {
	store.RegisterAdapter(adapterName, &adapter{})
}
