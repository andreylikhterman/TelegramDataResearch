package PrintChannels

import (
	"context"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain/PublicChannel"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

// PrintChannels получает и выводит историю постов и, если присутствует, комментарии (discussion) канала.
func PrintChannels(ctx context.Context, client *telegram.Client, log *zap.Logger, ch PublicChannel.PublicChannel) {
	// Получаем историю постов канала.
	posts, err := client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  ch.ID,
			AccessHash: ch.AccessHash,
		},
		Limit: 10,
	})
	if err != nil {
		log.Error("Failed to get channel posts", zap.String("channel", ch.Title), zap.Error(err))
	} else {
		var messages []tg.MessageClass
		switch p := posts.(type) {
		case *tg.MessagesMessages:
			messages = p.Messages
		case *tg.MessagesChannelMessages:
			messages = p.Messages
		default:
			log.Error("Unknown posts response type", zap.Any("response", p))
		}
		for _, msg := range messages {
			message, ok := msg.(*tg.Message)
			if !ok || message.Message == "" {
				continue
			}
			log.Info("Channel post",
				zap.String("channel", ch.Title),
				zap.String("text", message.Message),
				zap.Time("date", time.Unix(int64(message.Date), 0)),
			)
		}
	}

	// Получаем полную информацию о канале.
	fullResp, err := client.API().ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  ch.ID,
		AccessHash: ch.AccessHash,
	})
	if err != nil {
		log.Error("Failed to get full channel info", zap.String("channel", ch.Title), zap.Error(err))
		return
	}

	// Приводим ответ к *tg.MessagesChatFull.
	fullData := fullResp

	// Извлекаем ID обсуждения из объекта FullChat.
	var linkedChatID int64
	if fc, ok := fullData.FullChat.(*tg.ChannelFull); ok {
		// В зависимости от версии библиотеки поле может называться DiscussionGroupID или LinkedChatID.
		linkedChatID = fc.LinkedChatID
	} else {
		log.Info("No discussion group for channel", zap.String("channel", ch.Title))
		return
	}
	if linkedChatID == 0 {
		log.Info("No discussion group for channel", zap.String("channel", ch.Title))
		return
	}

	// Ищем объект обсуждения в списке чатов, чтобы получить access hash (если требуется).
	var discussionPeer tg.InputPeerClass
	for _, chat := range fullData.Chats {
		switch c := chat.(type) {
		case *tg.Channel:
			if c.ID == linkedChatID {
				// Для супергруппы используем InputPeerChannel.
				discussionPeer = &tg.InputPeerChannel{
					ChannelID:  c.ID,
					AccessHash: c.AccessHash,
				}
			}
		case *tg.Chat:
			if c.ID == linkedChatID {
				// Если обсуждение оказалось обычным чатом.
				discussionPeer = &tg.InputPeerChat{
					ChatID: c.ID,
				}
			}
		}
	}
	if discussionPeer == nil {
		log.Error("Discussion peer not found", zap.Int64("linkedChatID", linkedChatID))
		return
	}

	// Запрашиваем историю комментариев из обсуждения.
	comments, err := client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  discussionPeer,
		Limit: 10,
	})
	if err != nil {
		log.Error("Failed to get discussion comments", zap.String("channel", ch.Title), zap.Error(err))
		return
	}

	var commMessages []tg.MessageClass
	switch c := comments.(type) {
	case *tg.MessagesMessages:
		commMessages = c.Messages
	case *tg.MessagesChannelMessages:
		commMessages = c.Messages
	default:
		log.Error("Unknown comments response type", zap.Any("response", c))
		return
	}
	for _, msg := range commMessages {
		message, ok := msg.(*tg.Message)
		if !ok || message.Message == "" {
			continue
		}
		log.Info("Discussion comment",
			zap.String("channel", ch.Title),
			zap.String("text", message.Message),
			zap.Time("date", time.Unix(int64(message.Date), 0)),
		)
	}
}
