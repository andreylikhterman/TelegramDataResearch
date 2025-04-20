package application

import (
	"context"
	"log"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
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
			continue
		}
		time.Sleep(5 * time.Second)
	}

	return nil
}
