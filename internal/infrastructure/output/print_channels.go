package output

import (
	"context"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

// PrintChannels получает и выводит историю постов и комментарии (если есть) канала.
func PrintChannels(ctx context.Context, client *telegram.Client, log *zap.Logger, ch domain.PublicChannel) {
	printChannelPosts(ctx, client, log, ch)
	linkedChatID := getLinkedChatID(ctx, client, log, ch)

	if linkedChatID == 0 {
		return
	}

	discussionPeer := findDiscussionPeer(ctx, client, log, ch, linkedChatID)

	if discussionPeer == nil {
		return
	}

	printDiscussionComments(ctx, client, log, ch, discussionPeer)
}

func printChannelPosts(ctx context.Context, client *telegram.Client, log *zap.Logger, ch domain.PublicChannel) {
	posts, err := client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  ch.ID,
			AccessHash: ch.AccessHash,
		},
		Limit: 10,
	})
	if err != nil {
		log.Error("Failed to get channel posts", zap.String("channel", ch.Title), zap.Error(err))
		return
	}

	var messages []tg.MessageClass
	switch p := posts.(type) {
	case *tg.MessagesMessages:
		messages = p.Messages
	case *tg.MessagesChannelMessages:
		messages = p.Messages
	default:
		log.Error("Unknown posts response type", zap.Any("response", p))
		return
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

func getLinkedChatID(ctx context.Context, client *telegram.Client, log *zap.Logger, ch domain.PublicChannel) int64 {
	fullResp, err := client.API().ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  ch.ID,
		AccessHash: ch.AccessHash,
	})
	if err != nil {
		log.Error("Failed to get full channel info", zap.String("channel", ch.Title), zap.Error(err))
		return 0
	}

	if fc, ok := fullResp.FullChat.(*tg.ChannelFull); ok {
		return fc.LinkedChatID
	}

	log.Info("No discussion group for channel", zap.String("channel", ch.Title))

	return 0
}

func findDiscussionPeer(ctx context.Context, client *telegram.Client, log *zap.Logger,
	ch domain.PublicChannel, linkedChatID int64) tg.InputPeerClass {
	fullResp, err := client.API().ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  ch.ID,
		AccessHash: ch.AccessHash,
	})
	if err != nil {
		log.Error("Failed to get full channel info", zap.String("channel", ch.Title), zap.Error(err))
		return nil
	}

	for _, chat := range fullResp.Chats {
		switch c := chat.(type) {
		case *tg.Channel:
			if c.ID == linkedChatID {
				return &tg.InputPeerChannel{ChannelID: c.ID, AccessHash: c.AccessHash}
			}
		case *tg.Chat:
			if c.ID == linkedChatID {
				return &tg.InputPeerChat{ChatID: c.ID}
			}
		}
	}

	log.Error("Discussion peer not found", zap.Int64("linkedChatID", linkedChatID))

	return nil
}

func printDiscussionComments(ctx context.Context, client *telegram.Client,
	log *zap.Logger, ch domain.PublicChannel, discussionPeer tg.InputPeerClass) {
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
