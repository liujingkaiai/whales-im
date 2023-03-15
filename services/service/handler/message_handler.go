package handler

import (
	"im/services/service/database"
	"time"

	redis "github.com/go-redis/redis/v7"
	iris "github.com/kataras/iris/v12"
	"github.com/klintcheng/kim/wire"
	"github.com/klintcheng/kim/wire/rpc"
	"gorm.io/gorm"
)

type ServiceHandler struct {
	BaseDb    *gorm.DB
	MessageDb *gorm.DB
	Cache     *redis.Client
	Idgen     *database.IDGenerator
}

func (h *ServiceHandler) InsertUserMessage(c iris.Context) {
	var req rpc.InsertMessageReq
	if err := c.ReadBody(&req); err != nil {
		c.StopWithError(iris.StatusBadRequest, err)
		return
	}

	messageId := h.GenarateMessageId()
	messageContent := database.MessageContent{
		ID:       messageId,
		Type:     byte(req.Message.Type),
		Body:     req.Message.Body,
		Extra:    req.Message.Extra,
		SendTime: req.SendTime,
	}

	//扩散写
	idx := make([]database.MessageIndex, 2)
	idx[0] = database.MessageIndex{
		ID:        h.Idgen.Next().Int64(),
		MessageID: messageId,
		AccountA:  req.Dest,
		AccountB:  req.Sender,
		Direction: 0,
		SendTime:  req.SendTime,
	}
	idx[1] = database.MessageIndex{
		ID:        h.Idgen.Next().Int64(),
		MessageID: messageId,
		AccountB:  req.Dest,
		AccountA:  req.Sender,
		Direction: 1,
		SendTime:  req.SendTime,
	}
	err := h.MessageDb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&messageContent).Error; err != nil {
			return err
		}
		if err := tx.Create(&idx).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	_, _ = c.Negotiate(&rpc.InsertMessageResp{
		MessageId: messageId,
	})
}

func (h *ServiceHandler) InsertGroupMessage(c iris.Context) {
	var req rpc.InsertMessageReq
	if err := c.ReadBody(&req); err != nil {
		c.StopWithError(iris.StatusBadRequest, err)
		return
	}
	messageId := h.GenarateMessageId()
	var memebers []*database.GroupMember
	if err := h.BaseDb.Where(&database.GroupMember{Group: req.Dest}).Find(&memebers).Error; err != nil {
		c.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	//扩散写
	var ids = make([]database.MessageIndex, len(memebers))
	for i, m := range memebers {
		ids[i] = database.MessageIndex{
			ID:        h.GenarateMessageId(),
			MessageID: m.ID,
			AccountA:  m.Account,
			AccountB:  req.Sender,
			Direction: 0,
			Group:     m.Group,
			SendTime:  req.SendTime,
		}

		if m.Account == req.Sender {
			ids[i].Direction = 1
		}
	}

	messageContent := database.MessageContent{
		ID:       messageId,
		Type:     byte(req.Message.Type),
		Body:     req.Message.Body,
		Extra:    req.Message.Extra,
		SendTime: req.SendTime,
	}

	if err := h.MessageDb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&messageContent).Error; err != nil {
			return err
		}

		if err := tx.Create(&ids).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.StopWithError(iris.StatusInternalServerError, err)
		return
	}
	_, _ = c.Negotiate(&rpc.InsertMessageResp{
		MessageId: messageId,
	})
}

func (h *ServiceHandler) MessageAck(c iris.Context) {
	var req rpc.AckMessageReq
	if err := c.ReadBody(&req); err != nil {
		c.StopWithError(iris.StatusBadRequest, err)
		return
	}

	if err := setMessageAck(h.Cache, req.Account, req.MessageId); err != nil {
		c.StopWithError(iris.StatusInternalServerError, err)
		return
	}
}

func setMessageAck(cache *redis.Client, account string, messageID int64) error {
	if messageID == 0 {
		return nil
	}

	key := database.KeyMessageAckIndex(account)
	return cache.Set(key, messageID, wire.OfflineReadIndexExpiresIn).Err()
}

func (h *ServiceHandler) GetOfflineMessageIndex(c iris.Context) {
	var req rpc.GetOfflineMessageIndexReq
	if err := c.ReadBody(&req); err != nil {
		c.StopWithError(iris.StatusBadRequest, err)
		return
	}
	msgId := req.MessageId
	start, err := h.getSentTime(req.Account, msgId)
	if err != nil {
		c.StopWithError(iris.StatusInternalServerError, err)
		return
	}
	var indexes []*rpc.MessageIndex
	tx := h.MessageDb.Model(&database.MessageIndex{}).Select("send_time", "account_b", "direction", "message_id", "group")
	err = tx.Where("account_a=? and send_time>? and direction", req.Account, start, 0).Order("send_time asc").Limit(wire.OfflineSyncIndexCount).Find(&indexes).Error
	if err != nil {
		c.StopWithError(iris.StatusInternalServerError, err)
		return
	}
	err = setMessageAck(h.Cache, req.Account, req.MessageId)
	if err != nil {
		c.StopWithError(iris.StatusInternalServerError, err)
		return
	}
	c.Negotiate(&rpc.GetOfflineMessageIndexResp{
		List: indexes,
	})
}

func (h *ServiceHandler) GetOfflineMessageContent(c iris.Context) {
	var req rpc.GetOfflineMessageContentReq
	if err := c.ReadBody(&req); err != nil {
		c.StopWithError(iris.StatusBadRequest, err)
		return
	}
	mlen := len(req.MessageIds)
	if mlen > wire.MessageMaxCountPerPage {
		c.StopWithText(iris.StatusBadRequest, "too many MessageIds")
		return
	}
	contents := make([]*rpc.Message, 0, mlen)
	err := h.MessageDb.Model(&database.MessageContent{}).Where(req.MessageIds).Find(&contents).Error
	if err != nil {
		c.StopWithError(iris.StatusInternalServerError, err)
		return
	}
}

func (h *ServiceHandler) getSentTime(account string, msgID int64) (int64, error) {
	// 1.冷启动，从服务端拉取消息
	if msgID == 0 {
		key := database.KeyMessageAckIndex(account)
		msgID, _ = h.Cache.Get(key).Int64()
	}
	var start int64
	if msgID > 0 {
		//2.根据消息ID读取此条消息发送的时间
		var content database.MessageContent
		err := h.MessageDb.Select("send_time").First(&content, msgID).Error
		if err != nil {
			// 3.消息不存在，返回最近一天消息\
			time.Now().Unix()
			start = time.Now().AddDate(0, 0, -1).UnixNano()
		} else {
			start = content.SendTime
		}
	}
	//4.返回默认离线消息的过期时间
	earliestKeepTime := time.Now().AddDate(0, 0, -1*wire.OfflineMessageExpiresIn).UnixNano()
	if start == 0 || start < earliestKeepTime {
		start = earliestKeepTime
	}
	return start, nil
}

func (h *ServiceHandler) GenarateMessageId() int64 {
	return h.Idgen.Next().Int64()
}
