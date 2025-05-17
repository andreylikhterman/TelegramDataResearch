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
	comment, commentID, postID, data, repliedTo := extractMessageDetails(message)

	if userID == 0 {
		chPosts <- domain.Post{
			Text:        comment,
			PostId:      commentID,
			ChannelId:   channelPeer.ChannelID,
			ChannelName: channelName,
			Data:        data,
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
			Data:        data,
			RepliedTo:   repliedTo,
		}
	}
	return nil
}

func GetHistory(channel domain.PublicChannel, lastStoredID, messageID int64, chPosts chan domain.Post, chMessages chan domain.Message, repo *db.MyDB, client *telegram.Client, e tg.Entities) error {
	ctx := context.Background()

	if lastStoredID >= messageID {
		return nil
	}
	time.Sleep(5 * time.Second)
	const batchSize = 100

	for i := lastStoredID + 1; i <= messageID; i += batchSize {
		endID := min(i+batchSize-1, messageID)

		messages, err := fetchMessages(ctx, client, channel.DiscussionPeer, i, endID)
		if err != nil {
			log.Printf("Failed to fetch messages: %v", err)
			return err
		}

		handleFetchedMessages(ctx, messages, e, client, chPosts, chMessages)
	}

	return nil
}

func fetchMessages(ctx context.Context, client *telegram.Client, discussion tg.InputPeerClass, startID, endID int64) (tg.MessagesMessagesClass, error) {
	return client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  discussion,
		MaxID: int(endID),
		MinID: int(startID),
	})
}

func handleFetchedMessages(ctx context.Context, messages tg.MessagesMessagesClass, e tg.Entities, client *telegram.Client, chPosts chan domain.Post, chMessages chan domain.Message) {
	switch msg := messages.(type) {
	case *tg.MessagesChannelMessages:
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
