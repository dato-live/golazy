package main

import (
	"encoding/json"
	"github.com/dato-live/golazy/server/protos"
	"time"
)

func PBClientHiSerialize(msg *DMClientHi) *golazy.ClientMsg_Hi {
	return &golazy.ClientMsg_Hi{
		Hi: &golazy.ClientHi{
			ClientID:          msg.ClientID,
			ClientName:        msg.ClientName,
			ClientVersion:     msg.ClientVersion,
			ClientDescription: msg.ClientDescription,
			AllowedCommandIDs: msg.AllowedCommandIDs,
			Timestamp:         timeToInt64(msg.Timestamp),
		}}
}

func PBClientLeaveSerialize(msg *DMClientLeave) *golazy.ClientMsg_Leave {
	return &golazy.ClientMsg_Leave{
		Leave: &golazy.ClientLeave{
			ClientID:  msg.ClientID,
			Timestamp: timeToInt64(msg.Timestamp),
		}}
}

func PBClientReqSerialize(msg *DMClientReq) *golazy.ClientMsg_Req {
	return &golazy.ClientMsg_Req{
		Req: &golazy.ClientReq{
			ReqID:     msg.ReqID,
			From:      msg.From,
			To:        msg.To,
			CommandID: msg.CommandID,
			Content:   msg.Content,
			Timestamp: timeToInt64(msg.Timestamp),
		}}
}

func PBClientRespSerialize(msg *DMClientResp) *golazy.ClientMsg_Resp {
	return &golazy.ClientMsg_Resp{
		Resp: &golazy.ClientResp{
			RespID:    msg.RespID,
			From:      msg.From,
			To:        msg.To,
			Content:   msg.Content,
			ErrCode:   msg.ErrCode,
			ErrMsg:    msg.ErrMsg,
			Timestamp: timeToInt64(msg.Timestamp),
		}}
}

func PBAckMsgSerialize(msg *DMAckMsg) *golazy.ClientMsg_Ack {
	return &golazy.ClientMsg_Ack{
		Ack: &golazy.AckMsg{
			MsgID:     msg.MsgID,
			IsOk:      msg.IsOk,
			Msg:       msg.Msg,
			Timestamp: timeToInt64(msg.Timestamp),
		}}
}

func PbSerialize(msg *DMClientMsg) *golazy.ClientMsg {
	var pkt golazy.ClientMsg

	switch {
	case msg.Hi != nil:
		pkt.Message = PBClientHiSerialize(msg.Hi)
	case msg.Leave != nil:
		pkt.Message = PBClientLeaveSerialize(msg.Leave)
	case msg.Req != nil:
		pkt.Message = PBClientReqSerialize(msg.Req)
	case msg.Resp != nil:
		pkt.Message = PBClientRespSerialize(msg.Resp)
	case msg.Ack != nil:
		pkt.Message = PBAckMsgSerialize(msg.Ack)
	}
	pkt.MsgID = msg.MsgID

	return &pkt
}

func PbDeserialize(pkt *golazy.ClientMsg) *DMClientMsg {
	var msg DMClientMsg
	if hi := pkt.GetHi(); hi != nil {
		msg.Hi = &DMClientHi{
			ClientID:          hi.GetClientID(),
			ClientName:        hi.GetClientName(),
			ClientVersion:     hi.GetClientVersion(),
			ClientDescription: hi.GetClientDescription(),
			AllowedCommandIDs: hi.GetAllowedCommandIDs(),
			Timestamp:         int64ToTime(hi.GetTimestamp()),
		}
	} else if leave := pkt.GetLeave(); leave != nil {
		msg.Leave = &DMClientLeave{
			ClientID:  leave.GetClientID(),
			Timestamp: int64ToTime(leave.GetTimestamp()),
		}
	} else if req := pkt.GetReq(); req != nil {
		msg.Req = &DMClientReq{
			ReqID:     req.GetReqID(),
			From:      req.GetFrom(),
			To:        req.GetTo(),
			CommandID: req.GetCommandID(),
			Content:   req.GetContent(),
			Timestamp: int64ToTime(req.GetTimestamp()),
		}
	} else if resp := pkt.GetResp(); resp != nil {
		msg.Resp = &DMClientResp{
			RespID:    resp.GetRespID(),
			From:      resp.GetFrom(),
			To:        resp.GetTo(),
			Content:   resp.GetContent(),
			ErrCode:   resp.GetErrCode(),
			ErrMsg:    resp.GetErrMsg(),
			Timestamp: int64ToTime(resp.GetTimestamp()),
		}
	} else if ack := pkt.GetAck(); ack != nil {
		msg.Ack = &DMAckMsg{
			MsgID:     ack.GetMsgID(),
			IsOk:      ack.GetIsOk(),
			Msg:       ack.GetMsg(),
			Timestamp: int64ToTime(ack.GetTimestamp()),
		}
	}

	msg.MsgID = pkt.GetMsgID()
	return &msg
}

func timeToInt64(ts *time.Time) int64 {
	if ts != nil {
		return ts.UnixNano() / int64(time.Millisecond)
	}
	return 0
}

func int64ToTime(ts int64) *time.Time {
	if ts > 0 {
		res := time.Unix(ts/1000, ts%1000).UTC()
		return &res
	}
	return nil
}

func GetJsonString(obj interface{}) string {
	js, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(js)
}
