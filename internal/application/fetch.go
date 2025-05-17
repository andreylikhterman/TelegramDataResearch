package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

func FetchChannelDataByNames(ctx context.Context, client *telegram.Client, channelNames []string) ([]domain.PublicChannel, error) {
	var result []domain.PublicChannel
	for _, name := range channelNames {
		channel, discussionPeer, participantsCount, channelType, err := resolveAndFetchChannel(ctx, client, name)
		if err != nil {
			log.Printf("Не удалось обработать канал %s: %v", name, err)
			continue
		}

		result = append(result, domain.PublicChannel{
			ID:             channel.ID,
			AccessHash:     channel.AccessHash,
			Title:          name,
			DiscussionPeer: discussionPeer,
			SubsCount:      int64(participantsCount),
			Type:           channelType,
		})
		time.Sleep(time.Second * 1)
	}
	return result, nil
}

func FetchChannelDataByID(ctx context.Context, client *telegram.Client, channelIDs []int) ([]domain.PublicChannel, error) {
	var result []domain.PublicChannel
	for _, id := range channelIDs {

		channel, discussionPeer, participantsCount, channelType, err := fetchChannelByID(ctx, client, int64(id))
		if err != nil {
			log.Printf("Не удалось обработать канал %d: %v", id, err)
			continue
		}

		result = append(result, domain.PublicChannel{
			ID:             channel.ID,
			AccessHash:     channel.AccessHash,
			Title:          channel.Title,
			DiscussionPeer: discussionPeer,
			SubsCount:      int64(participantsCount),
			Type:           channelType,
		})
		time.Sleep(time.Second * 1)
	}
	return result, nil
}

func resolveAndFetchChannel(ctx context.Context, client *telegram.Client, username string) (*tg.Channel, *tg.InputPeerChannel, int, string, error) {
	api := client.API()
	res, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
	if err != nil {
		return nil, nil, 0, "", err
	}

	var channel *tg.Channel
	if peer, ok := res.Peer.(*tg.PeerChannel); ok {
		for _, chat := range res.Chats {
			if ch, ok := chat.(*tg.Channel); ok && ch.ID == peer.ChannelID {
				channel = ch
				break
			}
		}
	}

	if channel == nil || !channel.Broadcast {
		return nil, nil, 0, "", ErrChannelNotFound
	}

	return fetchFullChannelInfo(ctx, api, channel)
}

func fetchChannelByID(ctx context.Context, client *telegram.Client, id int64) (*tg.Channel, *tg.InputPeerChannel, int, string, error) {
	api := client.API()
	channelsResp, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{&tg.InputChannel{ChannelID: id}})
	if err != nil || len(channelsResp.GetChats()) == 0 {
		return nil, nil, 0, "", err
	}

	channel := channelsResp.GetChats()[0].(*tg.Channel)
	return fetchFullChannelInfo(ctx, api, channel)
}

func fetchFullChannelInfo(ctx context.Context, api *tg.Client, channel *tg.Channel) (*tg.Channel, *tg.InputPeerChannel, int, string, error) {
	input := &tg.InputChannel{
		ChannelID:  channel.ID,
		AccessHash: channel.AccessHash,
	}

	full, err := api.ChannelsGetFullChannel(ctx, input)
	if err != nil {
		return nil, nil, 0, "", err
	}

	cf, ok := full.FullChat.(*tg.ChannelFull)
	if !ok || cf.LinkedChatID == 0 {
		return nil, nil, 0, "", ErrNoDiscussionChat
	}

	var linkedChat *tg.Channel
	for _, chat := range full.Chats {
		if ch, ok := chat.(*tg.Channel); ok && ch.ID == cf.LinkedChatID {
			linkedChat = ch
			break
		}
	}

	if linkedChat == nil {
		return nil, nil, 0, "", ErrLinkedChatNotFound
	}

	discussionPeer := &tg.InputPeerChannel{
		ChannelID:  linkedChat.ID,
		AccessHash: linkedChat.AccessHash,
	}
	var channelType string
	if channel.Megagroup {
		channelType = "supergroup"
	} else if channel.Broadcast {
		if channel.Username != "" {
			channelType = "public"
		} else {
			channelType = "private"
		}
	} else {
		channelType = "unknown"
	}

	return channel, discussionPeer, cf.ParticipantsCount, channelType, nil
}

var (
	ErrChannelNotFound    = fmt.Errorf("канал не найден или не является broadcast")
	ErrNoDiscussionChat   = fmt.Errorf("отсутствует связанный чат обсуждений")
	ErrLinkedChatNotFound = fmt.Errorf("связанный чат не найден в Chats")
)
