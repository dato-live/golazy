package types

import (
	"time"
)

func TimeNow() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}

func GetExpiresTime(minute int) time.Time {
	m, _ := time.ParseDuration("1m")
	return time.Now().Add(m * time.Duration(minute))
}

const DefaultMsgVersion = "v1.0"
const DefaultMaxRetryCount = 100
const DefaultRetrySecondInterval = 30
const DefaultCleanDbMinuteInterval = 15

// 会话Session超时时间间隔
const DefaultIdleSessionTimeoutSecond = 55
const DefaultMaxMessageSize = 20971520
const DefaultMessageExpireMinuteInterval = 600
const StatusQueued = "Queued"
const StatusSucceeded = "Succeeded"
const StatusFailed = "Failed"
const StatusRetry = "Retrying"

//键值型元数据记录表
type KvMeta struct {
	KeyName  string `xorm:"varchar(32) notnull unique index pk 'key_name'"`
	KeyValue string `xorm:"text 'key_value'"`
}

type ReqReceived struct {
	Id        int64     `xorm:"int(11) pk notnull autoincr 'id'"`
	Version   string    `xorm:"varchar(32) 'version'"`
	MsgID     string    `xorm:"varchar(128) index notnull 'msg_id'"`
	ReqID     string    `xorm:"varchar(123) index notnull 'req_id'"`
	From      string    `xorm:"varchar(128) index notnull 'msg_from'"`
	To        string    `xorm:"varchar(128) index notnull 'msg_to'"`
	Content   string    `xorm:"mediumtext 'content'"`
	Added     time.Time `xorm:"datetime created 'added_time'"`
	ExpiresAt time.Time `xorm:"datetime 'expires_at'"`
	Retries   int       `xorm:"'retries'"`
	Status    string    `xorm:"varchar(32) index 'status'"`
}

type RespReceived struct {
	Id        int64     `xorm:"int(11) pk notnull autoincr 'id'"`
	Version   string    `xorm:"varchar(32) 'version'"`
	MsgID     string    `xorm:"varchar(128) index notnull 'msg_id'"`
	RespID    string    `xorm:"varchar(123) index notnull 'resp_id'"`
	From      string    `xorm:"varchar(128) index notnull 'msg_from'"`
	To        string    `xorm:"varchar(128) index notnull 'msg_to'"`
	Content   string    `xorm:"mediumtext 'content'"`
	Added     time.Time `xorm:"datetime created 'added_time'"`
	ExpiresAt time.Time `xorm:"datetime 'expires_at'"`
	Retries   int       `xorm:"'retries'"`
	Status    string    `xorm:"varchar(32) index 'status'"`
}
