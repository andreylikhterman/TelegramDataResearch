package application

import (
	"context"
	"fmt"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

func registerHandlers(dispatcher *tg.UpdateDispatcher, client *telegram.Client, ch_posts chan domain.Post, ch_messages chan domain.Message) {
	dispatcher.OnNewChannelMessage(handleNewChannelMessage(client, ch_posts, ch_messages))
}

func handleNewChannelMessage(client *telegram.Client, ch_posts chan domain.Post, ch_messages chan domain.Message) func(context.Context, tg.Entities, *tg.UpdateNewChannelMessage) error {
	return func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		message, ok := u.Message.AsNotEmpty()
		if !ok {
			return nil
		}

		channelPeer, ok := message.GetPeerID().(*tg.PeerChannel)
		if !ok {
			return nil
		}

		channelName := getChannelName(ctx, channelPeer.ChannelID, e, client)
		userID, username := getUser(ctx, message, e, client)
		comment, commentID, postID, data, replied_to := extractMessageDetails(message)

		if userID == 0 {
			ch_posts <- domain.Post{
				Text:        comment,
				PostId:      commentID,
				ChannelId:   channelPeer.ChannelID,
				ChannelName: channelName,
				Data:        data,
			}
		} else {
			ch_messages <- domain.Message{
				Text:        comment,
				CommentId:   commentID,
				PostId:      postID,
				UserId:      userID,
				ChannelName: channelName,
				UserName:    username,
				ChannelId:   channelPeer.ChannelID,
				Data:        data,
				RepliedTo:   replied_to,
			}
		}
		return nil
	}
}

func getChannelName(ctx context.Context, channelID int64, e tg.Entities, client *telegram.Client) string {
	if channel, exists := e.Channels[channelID]; exists && channel.Title != "" {
		return channel.Title
	}

	channels, err := client.API().ChannelsGetChannels(ctx, []tg.InputChannelClass{
		&tg.InputChannel{ChannelID: channelID},
	})
	if err != nil || len(channels.GetChats()) == 0 {
		return fmt.Sprintf("channel%d", channelID)
	}

	if channel, ok := channels.GetChats()[0].(*tg.Channel); ok {
		e.Channels[channelID] = channel
		if channel.Title != "" {
			return channel.Title
		}
	}

	return fmt.Sprintf("channel%d", channelID)
}

func extractMessageDetails(message tg.NotEmptyMessage) (string, int, int, time.Time, int) {
	msg, ok := message.(*tg.Message)
	if !ok {
		return "", 0, 0, time.Time{}, 0
	}
	repl := 0
	if msg.ReplyTo != nil {
		if reply, ok := msg.ReplyTo.(*tg.MessageReplyHeader); ok {
			repl = reply.ReplyToMsgID
		}
	}
	comment := msg.GetMessage()
	commentID := msg.GetID()
	data := msg.Date
	timestamp := int64(data) // например, UNIX-время
	t := time.Unix(timestamp, 0)
	postID := extractPostID(msg)

	return comment, commentID, postID, t, repl
}

func extractPostID(msg *tg.Message) int {
	if replyTo, ok := msg.GetReplyTo(); ok {
		if header, ok := replyTo.(*tg.MessageReplyHeader); ok {
			if topID, ok := header.GetReplyToTopID(); topID != 0 && ok {
				return topID
			}
			return header.ReplyToMsgID
		}
	}
	return 0
}

func getUser(ctx context.Context, message tg.NotEmptyMessage, e tg.Entities, client *telegram.Client) (int, string) {
	fromPeer, ok := message.GetFromID()
	if !ok {
		return 0, ""
	}

	userPeer, ok := fromPeer.(*tg.PeerUser)
	if !ok {
		return 0, ""
	}

	userID := userPeer.UserID
	if user, exists := e.Users[userID]; exists {
		return int(userID), nonEmptyUsername(user.Username)
	}

	return fetchAndCacheUser(ctx, userID, e, client)
}

func fetchAndCacheUser(ctx context.Context, userID int64, e tg.Entities, client *telegram.Client) (int, string) {
	users, err := client.API().UsersGetUsers(ctx, []tg.InputUserClass{
		&tg.InputUser{UserID: userID},
	})
	if err != nil || len(users) == 0 {
		return int(userID), ""
	}

	if fullUser, ok := users[0].(*tg.User); ok {
		e.Users[userID] = fullUser
		return int(userID), nonEmptyUsername(fullUser.Username)
	}

	return int(userID), ""
}

func nonEmptyUsername(username string) string {
	if username == "" {
		return "no username"
	}
	return username
}
