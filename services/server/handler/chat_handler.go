package handler

import (
	"errors"
	"fmt"
	"im/iface"
	"im/services/server/service"
	"time"

	"github.com/klintcheng/kim/wire/pkt"
	"github.com/klintcheng/kim/wire/rpc"
)

var ErrNoDestination = errors.New("dest is empty")

type ChatHandler struct {
	msgService   service.Message
	groupService service.Group
}

func NewChatHandler(message service.Message, group service.Group) *ChatHandler {
	return &ChatHandler{
		msgService:   message,
		groupService: group,
	}
}

func (h *ChatHandler) DoUserTalk(ctx iface.IContext) {
	fmt.Println("DoUserTalk ctx.Header().Dest:", ctx.Header().Dest)
	if ctx.Header().Dest == "" {
		ctx.RespWithError(pkt.Status_NoDestination, ErrNoDestination)
		return
	}

	//1.解包
	var req pkt.MessageReq
	if err := ctx.ReadBody(&req); err != nil {
		ctx.RespWithError(pkt.Status_InvalidPacketBody, err)
		return
	}
	//获取接收方信息
	receiver := ctx.Header().GetDest()
	loc, err := ctx.GetLocation(receiver, "")
	if err != nil && err != iface.ErrSessionNil {
		ctx.RespWithError(pkt.Status_SystemException, err)
		return
	}

	sendTime := time.Now().UnixNano()
	resp, err := h.msgService.InsertUser(ctx.Session().GetApp(), &rpc.InsertMessageReq{
		Sender:   ctx.Session().GetAccount(),
		Dest:     receiver,
		SendTime: sendTime,
		Message: &rpc.Message{
			Type:  req.GetType(),
			Body:  req.GetBody(),
			Extra: req.GetExtra(),
		},
	})
	if err != nil {
		ctx.RespWithError(pkt.Status_SystemException, err)
		return
	}
	msgId := resp.MessageId
	//如果接收方在线，发送消息
	if loc != nil {
		if err = ctx.Dispatch(&pkt.MessagePush{
			MessageId: msgId,
			Type:      req.GetType(),
			Body:      req.GetBody(),
			Extra:     req.GetExtra(),
			Sender:    ctx.Session().GetAccount(),
			SendTime:  sendTime,
		}, loc); err != nil {
			ctx.RespWithError(pkt.Status_SystemException, err)
			return
		}
	}

	//返回消息
	ctx.Resp(pkt.Status_Success, &pkt.MessageResp{
		MessageId: msgId,
		SendTime:  sendTime,
	})
}
