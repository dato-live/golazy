/******************************************************************************
 *
 *  Description :
 *
 *    Handler of gRPC connections. See also hdl_websock.go for websockets and
 *    hdl_longpoll.go for long polling.
 *
 *****************************************************************************/

package main

import (
	"fmt"
	"github.com/dato-live/golazy/server/protos"
	"github.com/dato-live/golazy/server/store"
	"github.com/dato-live/golazy/server/store/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io"
	"net"
	"time"
)

type grpcNodeServer struct {
}

func (sess *Session) closeGrpc() {
	if sess.proto == GRPC {
		sess.lock.Lock()
		sess.grpcNode = nil
		sess.lock.Unlock()
	}

	//连接断开，保存未发送的数据，同时删除当前会话Session
	//TODO:持久化存储信息队列
	globals.sessionStore.Delete(sess)
	logger.Warn(fmt.Sprintf("[Session Removed] Seesion: '%s' (ClientID=%s) has been removed, because grpc has closed !!!", sess.sid, sess.clientInfo.ClientID))

}

//

// Equivalent of starting a new session and a read loop in one
func (*grpcNodeServer) MessageLoop(stream golazy.Node_MessageLoopServer) error {
	sess, _ := globals.sessionStore.NewSession(stream, "")

	defer func() {
		sess.closeGrpc()
	}()

	go sess.writeGrpcLoop()

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			//GRPC连接断开
			logger.Warn("grpc: recv", zap.String("session", sess.sid), zap.Error(err))
			return err
		}
		logger.Debug(fmt.Sprintf("grpc in"), zap.String("in", in.String()), zap.String("session", sess.sid))
		sess.dispatchMsg(PbDeserialize(in))

		sess.lock.Lock()
		if sess.grpcNode == nil {
			sess.lock.Unlock()
			break
		}
		sess.lock.Unlock()
	}

	return nil
}

func (sess *Session) dispatchMsg(msg *DMClientMsg) {

	switch {
	case msg.Hi != nil:
		now := types.TimeNow()
		existClient := globals.sessionStore.GetByClientID(msg.Hi.ClientID)
		if existClient != nil {
			logger.Warn(fmt.Sprintf("[Duplicated Client] Client: '%s' already connected, this connection will be dropped!", msg.Hi.ClientID), zap.String("ClientID", msg.Hi.ClientID))
			sess.queueOut(&DMClientMsg{Ack: &DMAckMsg{MsgID: msg.MsgID, IsOk: false, Msg: fmt.Sprintf("Duplicated client, Client [%s] already connected", msg.Hi.ClientID), Timestamp: &now}})
			time.Sleep(2 * time.Second)
			sess.closeGrpc()
			return
		}
		sess.clientInfo.ClientID = msg.Hi.ClientID
		sess.clientInfo.ClientName = msg.Hi.ClientName
		sess.clientInfo.ClientVersion = msg.Hi.ClientVersion
		sess.clientInfo.ClientDescription = msg.Hi.ClientDescription
		sess.clientInfo.AllowedCommandIDs = msg.Hi.AllowedCommandIDs
		sess.queueOut(&DMClientMsg{Ack: &DMAckMsg{MsgID: msg.MsgID, IsOk: true, Msg: "OK", Timestamp: &now}})
	case msg.Leave != nil:
		sess.closeGrpc()

	case msg.Req != nil:
		reqReplyMsgID, _ := globals.sessionStore.uidGen.NewMsgUid()
		replyReqMsg := &DMClientMsg{
			Req: &DMClientReq{
				ReqID:     msg.Req.ReqID,
				From:      msg.Req.From,
				To:        msg.Req.To,
				CommandID: msg.Req.CommandID,
				Timestamp: msg.Req.Timestamp,
			},
			MsgID: reqReplyMsgID,
		}

		//查找发送到的目标
		reqToSess := globals.sessionStore.GetByClientID(msg.Req.To)
		if reqToSess != nil {
			//存储消息
			store.MsgObj.InsertReq(&types.ReqReceived{
				Version:   types.DefaultMsgVersion,
				MsgID:     replyReqMsg.MsgID,
				ReqID:     replyReqMsg.Req.ReqID,
				From:      replyReqMsg.Req.From,
				To:        replyReqMsg.Req.To,
				Content:   GetJsonString(replyReqMsg),
				ExpiresAt: types.GetExpiresTime(globals.configs.MessageExpireMinuteInterval),
				Retries:   0,
				Status:    types.StatusQueued,
			})

			reqToSess.queueOut(replyReqMsg)

			reqAckMsgID, _ := globals.sessionStore.uidGen.NewMsgUid()
			now := types.TimeNow()
			sess.queueOut(&DMClientMsg{
				Ack: &DMAckMsg{
					MsgID:     msg.MsgID,
					IsOk:      true,
					Msg:       "OK",
					Timestamp: &now,
				},
				MsgID: reqAckMsgID,
			})
		} else {
			store.MsgObj.InsertReq(&types.ReqReceived{
				Version:   types.DefaultMsgVersion,
				MsgID:     replyReqMsg.MsgID,
				ReqID:     replyReqMsg.Req.ReqID,
				From:      replyReqMsg.Req.From,
				To:        replyReqMsg.Req.To,
				Content:   GetJsonString(replyReqMsg),
				ExpiresAt: types.GetExpiresTime(globals.configs.MessageExpireMinuteInterval),
				Retries:   0,
				Status:    types.StatusFailed,
			})

			reqAckMsgID, _ := globals.sessionStore.uidGen.NewMsgUid()
			now := types.TimeNow()
			sess.queueOut(&DMClientMsg{
				Ack: &DMAckMsg{
					MsgID:     msg.MsgID,
					IsOk:      false,
					Msg:       fmt.Sprintf("Target Not Found, Please Online target [%s] first", msg.Req.To),
					Timestamp: &now,
				},
				MsgID: reqAckMsgID,
			})
		}

	case msg.Resp != nil:
		respReplyMsgID, _ := globals.sessionStore.uidGen.NewMsgUid()
		replyRespMsg := &DMClientMsg{
			Resp: &DMClientResp{
				RespID:    msg.Resp.RespID,
				From:      msg.Resp.From,
				To:        msg.Resp.To,
				Content:   msg.Resp.Content,
				ErrCode:   msg.Resp.ErrCode,
				ErrMsg:    msg.Resp.ErrMsg,
				Timestamp: msg.Resp.Timestamp,
			},
			MsgID: respReplyMsgID,
		}

		//查找接收结果的目标
		respToSess := globals.sessionStore.GetByClientID(msg.Resp.To)
		if respToSess != nil {

			store.MsgObj.InsertResp(&types.RespReceived{
				Version:   types.DefaultMsgVersion,
				MsgID:     replyRespMsg.MsgID,
				RespID:    replyRespMsg.Resp.RespID,
				From:      replyRespMsg.Resp.From,
				To:        replyRespMsg.Resp.To,
				Content:   GetJsonString(replyRespMsg),
				ExpiresAt: types.GetExpiresTime(globals.configs.MessageExpireMinuteInterval),
				Retries:   0,
				Status:    types.StatusQueued,
			})

			respToSess.queueOut(replyRespMsg)

			respAckMsgID, _ := globals.sessionStore.uidGen.NewMsgUid()
			now := types.TimeNow()
			sess.queueOut(&DMClientMsg{
				Ack: &DMAckMsg{
					MsgID:     msg.MsgID,
					IsOk:      true,
					Msg:       "OK",
					Timestamp: &now,
				},
				MsgID: respAckMsgID,
			})
		} else {

			store.MsgObj.InsertResp(&types.RespReceived{
				Version:   types.DefaultMsgVersion,
				MsgID:     replyRespMsg.MsgID,
				RespID:    replyRespMsg.Resp.RespID,
				From:      replyRespMsg.Resp.From,
				To:        replyRespMsg.Resp.To,
				Content:   GetJsonString(replyRespMsg),
				ExpiresAt: types.GetExpiresTime(globals.configs.MessageExpireMinuteInterval),
				Retries:   0,
				Status:    types.StatusFailed,
			})

			respAckMsgID, _ := globals.sessionStore.uidGen.NewMsgUid()
			now := types.TimeNow()
			sess.queueOut(&DMClientMsg{
				Ack: &DMAckMsg{
					MsgID:     msg.MsgID,
					IsOk:      false,
					Msg:       fmt.Sprintf("Target Not Found, Please Online target [%s] first", msg.Req.To),
					Timestamp: &now,
				},
				MsgID: respAckMsgID,
			})
		}
	case msg.Ack != nil:
		logger.Debug(fmt.Sprintf("Client Ack Msg: %v", msg.Ack))

	}

}

func (sess *Session) writeGrpcLoop() {

	defer func() {
		sess.closeGrpc() // exit MessageLoop
	}()

	for {
		select {
		case msg, ok := <-sess.send:
			if !ok {
				// channel closed
				return
			}
			statusType := "unknown"
			m := msg.(*golazy.ClientMsg)
			if n := m.GetReq(); n != nil {
				statusType = "req"
			} else if n := m.GetResp(); n != nil {
				statusType = "resp"
			}

			if err := grpcWrite(sess, msg); err != nil {
				logger.Error("grpc: write", zap.String("session", sess.sid), zap.Error(err))
				sess.msgSendStatus <- MsgSendStatus{MsgID: m.MsgID, MsgType: statusType, IsOk: false}
				return
			}
			sess.msgSendStatus <- MsgSendStatus{MsgID: m.MsgID, MsgType: statusType, IsOk: true}

		case msg := <-sess.stop:
			// Shutdown requested, don't care if the message is delivered
			if msg != nil {
				err := grpcWrite(sess, msg)
				logger.Error("grpc: stop", zap.String("session", sess.sid), zap.Error(err))
				return
			}
			return

		case msgStatus := <-sess.msgSendStatus:
			switch msgStatus.MsgType {
			case "req":
				msg, err := store.MsgObj.GetReqByMsgID(msgStatus.MsgID)
				if err != nil {
					logger.Error("GetReqByMsgID failed", zap.String("MsgID", msgStatus.MsgID), zap.Error(err))
				} else {
					if msgStatus.IsOk {
						msg.Status = types.StatusSucceeded
					} else {
						msg.Status = types.StatusFailed
					}

					err = store.MsgObj.UpdateReq(msg)
					if err != nil {
						logger.Error("UpdateReq failed", zap.String("MsgID", msgStatus.MsgID), zap.Error(err))
					}
				}
			case "resp":
				msg, err := store.MsgObj.GetRespByMsgID(msgStatus.MsgID)
				if err != nil {
					logger.Error("GetRespByMsgID failed", zap.String("MsgID", msgStatus.MsgID), zap.Error(err))
				} else {
					if msgStatus.IsOk {
						msg.Status = types.StatusSucceeded
					} else {
						msg.Status = types.StatusFailed
					}

					err = store.MsgObj.UpdateResp(msg)
					if err != nil {
						logger.Error("UpdateResp failed", zap.String("MsgID", msgStatus.MsgID), zap.Error(err))
					}
				}
			}

		}
	}
}

func grpcWrite(sess *Session, msg interface{}) error {
	out := sess.grpcNode
	if out != nil {
		// Will panic if msg is not of *pbx.ServerMsg type. This is an intentional panic.
		return out.Send(msg.(*golazy.ClientMsg))
	}
	return nil
}

func serveGrpc(addr string) (*grpc.Server, error) {
	if addr == "" {
		return nil, nil
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	srv := grpc.NewServer(grpc.MaxRecvMsgSize(int(globals.configs.MaxMessageSize)))
	golazy.RegisterNodeServer(srv, &grpcNodeServer{})
	logger.Info(fmt.Sprintf("gRPC server is registered at [%s]", addr))

	go func() {
		if err := srv.Serve(lis); err != nil {
			logger.Error("gRPC server failed:", zap.Error(err))
		}
	}()

	return srv, nil
}
