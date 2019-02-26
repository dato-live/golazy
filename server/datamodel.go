package main

import "time"

type DMClientHi struct {
	ClientID          string           `json:"clientid"`
	ClientName        string           `json:"clientname"`
	ClientVersion     string           `json:"clientversion"`
	ClientDescription string           `json:"clientdescription"`
	AllowedCommandIDs map[int64]string `json:"allowedcommandids"`
	Timestamp         *time.Time       `json:"timestamp"`
}

type DMClientLeave struct {
	ClientID  string     `json:"clientid"`
	Timestamp *time.Time `json:"timestamp"`
}

type DMClientReq struct {
	ReqID     string     `json:"reqid"`
	From      string     `json:"from"`
	To        string     `json:"to"`
	CommandID int64      `json:"commandid"`
	Content   string     `json:"content"`
	Timestamp *time.Time `json:"timestamp"`
}

type DMClientResp struct {
	RespID    string     `json:"respid"`
	From      string     `json:"from"`
	To        string     `json:"to"`
	Content   string     `json:"content"`
	ErrCode   int32      `json:"errcode"`
	ErrMsg    string     `json:"errmsg"`
	Timestamp *time.Time `json:"timestamp"`
}

type DMAckMsg struct {
	MsgID     string     `json:"msgid"`
	IsOk      bool       `json:"isok"`
	Msg       string     `json:"msg"`
	Timestamp *time.Time `json:"timestamp"`
}

type DMClientMsg struct {
	Hi    *DMClientHi    `json:"hi,omitempty"`
	Leave *DMClientLeave `json:"leave,omitempty"`
	Req   *DMClientReq   `json:"req,omitempty"`
	Resp  *DMClientResp  `json:"resp,omitempty"`
	Ack   *DMAckMsg      `json:"ack,omitempty"`
	MsgID string         `json:"msgid"`
}
