package application

import (
	"context"
	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"log"
)

func FetchChannelDataByNames(ctx context.Context, client *telegram.Client, channelNames []string) ([]domain.PublicChannel, error) {
	api := client.API()
	var channels []domain.PublicChannel

	for _, name := range channelNames {
		// Разрешаем юзернейм
		req := &tg.ContactsResolveUsernameRequest{
			Username: name,
		}
		resolved, err := api.ContactsResolveUsername(ctx, req)
		if err != nil {
			log.Printf("Ошибка при разрешении юзернейма %s: %v", name, err)
			continue
		}

		// Ищем канал, соответствующий разрешенному Peer
		var channel *tg.Channel
		if peerChannel, ok := resolved.Peer.(*tg.PeerChannel); ok {
			for _, chat := range resolved.Chats {
				if ch, ok := chat.(*tg.Channel); ok && ch.ID == peerChannel.ChannelID {
					channel = ch
					break
				}
			}
		}
		if channel == nil || !channel.Broadcast {
			log.Printf("Broadcast-канал с юзернеймом %s не найден", name)
			continue
		}

		// Формируем входной объект для запроса полной информации
		inputChannel := &tg.InputChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		}

		//log.Print(channel.ID)
		//log.Print(channel.AccessHash)

		// Получаем полную информацию о канале
		full, err := api.ChannelsGetFullChannel(ctx, inputChannel)
		if err != nil {
			log.Printf("Ошибка получения полной информации для канала %s: %v", name, err)
			continue
		}

		msgChatFull := full
		cf, ok := msgChatFull.FullChat.(*tg.ChannelFull)
		if !ok {
			log.Printf("Неожиданный формат полной информации для канала %s", name)
			continue
		}

		// Проверяем наличие группы обсуждения
		if cf.LinkedChatID == 0 {
			log.Printf("У канала %s не найден связанный чат обсуждений", name)
			//	log.Printf(cf.String())
			continue
		}

		// Ищем связанный чат в списке Chats для получения AccessHash
		var linkedChat *tg.Channel
		for _, chat := range msgChatFull.Chats {
			if ch, ok := chat.(*tg.Channel); ok && ch.ID == cf.LinkedChatID {
				linkedChat = ch
				break
			}
		}
		if linkedChat == nil {
			log.Printf("Связанный чат с ID %d не найден в ответе для канала %s", cf.LinkedChatID, name)
			continue
		}

		// Формируем данные канала с корректным DiscussionPeer
		chData := domain.PublicChannel{
			ID:         channel.ID,
			AccessHash: channel.AccessHash,
			Title:      name,
			DiscussionPeer: &tg.InputPeerChannel{
				ChannelID:  linkedChat.ID,
				AccessHash: linkedChat.AccessHash,
			},
		}
		channels = append(channels, chData)
	}

	return channels, nil
}

func FetchChannelDataByID(ctx context.Context, client *telegram.Client, channelNames []int) ([]domain.PublicChannel, error) {
	api := client.API()
	var channels []domain.PublicChannel

	for _, id := range channelNames {
		inputChannelNoHash := &tg.InputChannel{
			ChannelID: int64(id),
		}

		channels_curr, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{inputChannelNoHash})

		if err != nil {
			log.Printf("failed to fetch channel %w", id)
			continue
		}

		if len(channels_curr.GetChats()) == 0 {
			log.Printf("no channel found %w", id)
			continue
		}

		channel := channels_curr.GetChats()[0].(*tg.Channel)

		inputChannel := &tg.InputChannel{
			ChannelID:  int64(id),
			AccessHash: channel.AccessHash,
		}

		// Получаем полную информацию о канале
		full, err := api.ChannelsGetFullChannel(ctx, inputChannel)
		if err != nil {
			log.Printf("Ошибка получения полной информации для канала %s: %v", id, err)
			continue
		}

		msgChatFull := full
		cf, ok := msgChatFull.FullChat.(*tg.ChannelFull)
		if !ok {
			log.Printf("Неожиданный формат полной информации для канала %s", id)
			continue
		}

		// Проверяем наличие группы обсуждения
		if cf.LinkedChatID == 0 {
			log.Printf("У канала %s не найден связанный чат обсуждений", id)
			//	log.Printf(cf.String())
			continue
		}

		// Ищем связанный чат в списке Chats для получения AccessHash
		var linkedChat *tg.Channel
		for _, chat := range msgChatFull.Chats {
			if ch, ok := chat.(*tg.Channel); ok && ch.ID == cf.LinkedChatID {
				linkedChat = ch
				break
			}
		}
		if linkedChat == nil {
			log.Printf("Связанный чат с ID %d не найден в ответе для канала %s", cf.LinkedChatID, id)
			continue
		}

		// Формируем данные канала с корректным DiscussionPeer
		chData := domain.PublicChannel{
			ID:         channel.ID,
			AccessHash: channel.AccessHash,
			Title:      channel.Title,
			DiscussionPeer: &tg.InputPeerChannel{
				ChannelID:  linkedChat.ID,
				AccessHash: linkedChat.AccessHash,
			},
		}
		channels = append(channels, chData)
	}

	return channels, nil
}
