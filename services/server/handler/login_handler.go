package handler

import (
	"fmt"
	"im/iface"
	"im/logger"

	"github.com/klintcheng/kim/wire/pkt"
)

type LoginHandler struct {
}

func NewLoginHandler() *LoginHandler {
	return &LoginHandler{}
}

func (h *LoginHandler) DoSysLogin(ctx iface.IContext) {
	var session pkt.Session
	if err := ctx.ReadBody(&session); err != nil {
		ctx.RespWithError(pkt.Status_InvalidPacketBody, err)
		return
	}
	logger.WithFields(logger.Fields{
		"Func":      "Login",
		"ChannelId": session.GetChannelId(),
		"Account":   session.GetAccount(),
		"RemoteIP":  session.GetRemoteIP(),
	}).Info("do login")

	//检测当前用户是否在其他地方登录
	old, err := ctx.GetLocation(session.Account, "")
	// fmt.Println(err != iface.ErrSessionNil)
	fmt.Printf("err != iface.ErrSessionNil 结果:%v \n", err != iface.ErrSessionNil)
	fmt.Println("err:", err)
	if err != nil && err != iface.ErrSessionNil {
		ctx.RespWithError(pkt.Status_SystemException, iface.ErrSessionNil)
		return
	}

	if old != nil {
		ctx.Dispatch(&pkt.KickoutNotify{ChannelId: old.ChannelID}, old)
	}
	// 添加会话到会话管理
	err = ctx.Add(&session)
	if err != nil {
		ctx.RespWithError(pkt.Status_SystemException, err)
		return
	}
	//通知登录成功
	resp := &pkt.LoginResp{
		ChannelId: session.ChannelId,
	}
	_ = ctx.Resp(pkt.Status_Success, resp)
}

func (h *LoginHandler) DoSysLogout(ctx iface.IContext) {
	logger.WithFields(logger.Fields{
		"Func":      "Logout",
		"ChannelId": ctx.Session().GetChannelId(),
		"Account":   ctx.Session().GetAccount(),
	}).Info("do Logout ")

	err := ctx.Delete(ctx.Session().GetAccount(), ctx.Session().GetChannelId())
	if err != nil {
		ctx.RespWithError(pkt.Status_SystemException, err)
		return
	}
	ctx.Resp(pkt.Status_Success, nil)
}
