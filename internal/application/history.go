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

func HandleMessage(ctx context.Context, newMessage *tg.Message, channelPost chan domain.Post, channelMessage chan domain.Message, client *telegram.Client, e tg.Entities) error {
	message, ok := newMessage.AsNotEmpty()
	if !ok {
		return nil
	}

	channelPeer, ok := message.GetPeerID().(*tg.PeerChannel)
	if !ok {
		return nil
	}

	channelName := getChannelName(ctx, channelPeer.ChannelID, e, client)

	userID, username := GetUser(ctx, message, e, client)
	comment, commentID, postID := extractMessageDetails(message)

	if userID == 0 {
		channelPost <- domain.Post{Text: comment, PostId: commentID, ChannelId: channelPeer.ChannelID,
			Channel: channelName}
	} else {
		channelMessage <- domain.Message{Text: comment, CommentId: commentID, PostId: postID, UserId: userID,
			ChannelName: channelName, UserName: username, ChannelId: channelPeer.ChannelID}
	}
	return nil
}

func GetHistory(channelID int64, lastStoredID int64, messageID int64, channelPost chan domain.Post, channelMessage chan domain.Message, repo *db.MyDB, client *telegram.Client, e tg.Entities) error {
	ctx := context.Background()

	if lastStoredID >= messageID {
		return nil
	}

	time.Sleep(5 * time.Second)
	req := &tg.ChannelsGetMessagesRequest{
		Channel: &tg.InputChannel{
			ChannelID: channelID,
		},
		ID: make([]tg.InputMessageClass, 0),
	}

	const batchSize = 100
	for i := lastStoredID; i <= messageID; i += batchSize {
		endID := i + batchSize - 1
		if endID > messageID {
			endID = messageID
		}

		req.ID = req.ID[:0]

		for j := i; j <= endID; j++ {
			req.ID = append(req.ID, &tg.InputMessageID{ID: int(j)})
		}

		messages, err := client.API().ChannelsGetMessages(ctx, req)
		if err != nil {
			log.Printf("Failed to fetch messages: %v", err)
			return err
		}

		switch msg := messages.(type) {
		case *tg.MessagesMessages:
			for _, m := range msg.Messages {
				if message, ok := m.(*tg.Message); ok {
					err := HandleMessage(ctx, message, channelPost, channelMessage, client, e)
					if err != nil {
						log.Printf("Failed to store message %d: %v", message.ID, err)
						continue
					}
				}
			}
		default:
			log.Printf("Unexpected message type: %T", msg)
			continue
		}
	}

	return nil
}
