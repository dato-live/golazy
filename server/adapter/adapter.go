//数据库适配器接口定义包
package adapter

import (
	"github.com/dato-live/golazy/server/config"
	"github.com/dato-live/golazy/server/store/types"
)

type Adapter interface {
	//打开并配置数据库适配器
	Open(conf config.Config) error
	//关闭适配器
	Close() error
	//检查适配器是否已经打开可用
	IsOpen() bool
	//检查数据库版本与当前适配器是否匹配
	CheckDbVersion() error
	//获取当前适配器名称
	GetName() string
	//创建数据库，若设置reset为true，则会先删除存在的数据库，再重新创建
	CreateDb(reset bool) error

	//消息传递持久化存储/读取/更新/删除操作接口
	InsertReq(received *types.ReqReceived) error
	InsertResp(received *types.RespReceived) error

	GetReqByMsgID(msgId string) (*types.ReqReceived, error)
	GetRespByMsgID(msgId string) (*types.RespReceived, error)

	UpdateReq(req *types.ReqReceived) error
	UpdateResp(resp *types.RespReceived) error

	DeleteReq(id int64) error
	DeleteResp(id int64) error

	DeleteSendedOrExpireMsg() error

	GetRetryReq() ([]types.ReqReceived, error)
	GetRetryResp() ([]types.RespReceived, error)
}
