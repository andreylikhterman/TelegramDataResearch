package application

import (
	"context"
	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"log"
	"time"
)

func SubscribeToDiscussionChats(ctx context.Context, client *telegram.Client, channels []domain.PublicChannel) error {
	api := client.API()

	for _, ch := range channels {
		peer, ok := ch.DiscussionPeer.(*tg.InputPeerChannel)
		if !ok {
			log.Printf("Связанный чат для канала %s не имеет типа *tg.InputPeerChannel", ch.Title)
			continue
		}

		inputChannel := &tg.InputChannel{
			ChannelID:  peer.ChannelID,
			AccessHash: peer.AccessHash,
		}

		_, err := api.ChannelsJoinChannel(ctx, inputChannel)
		if err != nil {
			log.Printf("Ошибка при вступлении в чат обсуждений для канала %s: %v", ch.Title, err)
			continue
		}
		log.Printf("Вступили в чат обсуждений канала %s", ch.Title)
		time.Sleep(5 * time.Second)
	}

	return nil
}
