package application

import (
	"context"
	"log"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/logger"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

func SubscribeToDiscussionChats(ctx context.Context, client *telegram.Client, channels []domain.PublicChannel, logger *logger.Logger) error {
	for _, ch := range channels {
		if err := joinDiscussionChannel(ctx, client, ch); err != nil {
			log.Printf("Не удалось присоединиться к чату %s: %v", ch.Title, err)
		}

		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}

	}
	return nil
}

func joinDiscussionChannel(ctx context.Context, client *telegram.Client, ch domain.PublicChannel) error {
	peer, ok := ch.DiscussionPeer.(*tg.InputPeerChannel)
	if !ok {
		log.Printf("Связанный чат для канала %s не имеет типа *tg.InputPeerChannel", ch.Title)
		return nil
	}

	inputChannel := &tg.InputChannel{
		ChannelID:  peer.ChannelID,
		AccessHash: peer.AccessHash,
	}

	_, err := client.API().ChannelsJoinChannel(ctx, inputChannel)
	return err
}
