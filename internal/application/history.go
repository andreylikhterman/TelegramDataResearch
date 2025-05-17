package application

import (
	"context"
	"log"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/db"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

func processMessage(ctx context.Context, message tg.NotEmptyMessage, e tg.Entities, client *telegram.Client, chPosts chan domain.Post, chMessages chan domain.Message) error {
	channelPeer, ok := message.GetPeerID().(*tg.PeerChannel)
	if !ok {
		return nil
	}

	channelName := getChannelName(ctx, channelPeer.ChannelID, e, client)
	userID, username := getUser(ctx, message, e, client)
	comment, commentID, postID := extractMessageDetails(message)

	if userID == 0 {
		chPosts <- domain.Post{
			Text:        comment,
			PostId:      commentID,
			ChannelId:   channelPeer.ChannelID,
			ChannelName: channelName,
		}
	} else {
		chMessages <- domain.Message{
			Text:        comment,
			CommentId:   commentID,
			PostId:      postID,
			UserId:      userID,
			ChannelName: channelName,
			UserName:    username,
			ChannelId:   channelPeer.ChannelID,
		}
	}
	return nil
}

func GetHistory(channelID, lastStoredID, messageID int64, chPosts chan domain.Post, chMessages chan domain.Message, repo *db.MyDB, client *telegram.Client, e tg.Entities) error {
	ctx := context.Background()

	if lastStoredID >= messageID {
		return nil
	}

	time.Sleep(5 * time.Second)
	const batchSize = 100

	for i := lastStoredID; i <= messageID; i += batchSize {
		endID := min(i+batchSize-1, messageID)
		ids := buildMessageIDList(i, endID)

		messages, err := fetchMessages(ctx, client, channelID, ids)
		if err != nil {
			log.Printf("Failed to fetch messages: %v", err)
			return err
		}

		handleFetchedMessages(ctx, messages, e, client, chPosts, chMessages)
	}

	return nil
}

func buildMessageIDList(start, end int64) []tg.InputMessageClass {
	ids := make([]tg.InputMessageClass, 0, end-start+1)
	for id := start; id <= end; id++ {
		ids = append(ids, &tg.InputMessageID{ID: int(id)})
	}
	return ids
}

func fetchMessages(ctx context.Context, client *telegram.Client, channelID int64, ids []tg.InputMessageClass) (tg.MessagesMessagesClass, error) {
	return client.API().ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: &tg.InputChannel{ChannelID: channelID},
		ID:      ids,
	})
}

func handleFetchedMessages(ctx context.Context, messages tg.MessagesMessagesClass, e tg.Entities, client *telegram.Client, chPosts chan domain.Post, chMessages chan domain.Message) {
	switch msg := messages.(type) {
	case *tg.MessagesMessages:
		for _, m := range msg.Messages {
			if message, ok := m.(*tg.Message); ok {
				if err := processMessage(ctx, message, e, client, chPosts, chMessages); err != nil {
					log.Printf("Failed to process message %d: %v", message.ID, err)
				}
			}
		}
	default:
		log.Printf("Unexpected message type: %T", msg)
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
