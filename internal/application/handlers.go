package application

import (
	"context"
	"fmt"

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

		userID, username := GetUser(ctx, message, e, client)
		comment, commentID, postID := extractMessageDetails(message)

		if userID == 0 {
			ch_posts <- domain.Post{Text: comment, PostId: commentID, ChannelId: channelPeer.ChannelID,
				Channel: channelName}
		} else {
			ch_messages <- domain.Message{Text: comment, CommentId: commentID, PostId: postID, UserId: userID,
				ChannelName: channelName, UserName: username, ChannelId: channelPeer.ChannelID}
		}
		return nil
	}
}

func getChannelName(ctx context.Context, channelID int64, e tg.Entities, client *telegram.Client) string {
	if channel, exists := e.Channels[channelID]; exists {
		if channel.Title != "" {
			return channel.Title
		}
		return fmt.Sprintf("channel%d", channelID)
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

func extractMessageDetails(message tg.NotEmptyMessage) (string, int, int) {
	var comment string
	var commentID, postID int

	info, ok := message.(*tg.Message)
	if !ok {
		return "false", 0, 0
	}

	comment = info.GetMessage()
	commentID = info.GetID()

	if replyTo, ok := info.GetReplyTo(); ok {
		if header, ok := replyTo.(*tg.MessageReplyHeader); ok {
			postID, ok = header.GetReplyToTopID()
			if postID == 0 {
				postID = header.ReplyToMsgID
			}
		}
	}
	return comment, commentID, postID
}

func GetUser(ctx context.Context, message tg.NotEmptyMessage, e tg.Entities, client *telegram.Client) (int, string) {
	fromPeer, ok := message.GetFromID()
	if !ok {
		return 0, ""
	}

	userPeer, ok := fromPeer.(*tg.PeerUser)
	if !ok {
		return 0, ""
	}

	var username string
	if user, exists := e.Users[userPeer.UserID]; exists {
		username = user.Username
		if username == "" {
			username = fmt.Sprint("no username")
		}
	} else {
		users, err := client.API().UsersGetUsers(ctx, []tg.InputUserClass{
			&tg.InputUser{UserID: userPeer.GetUserID()},
		})
		if err == nil && len(users) > 0 {
			if fullUser, ok := users[0].(*tg.User); ok {
				username = fullUser.Username
				if username == "" {
					username = fmt.Sprintf("no username")
				}
				e.Users[userPeer.UserID] = fullUser
			}
		}
	}
	return int(userPeer.UserID), username
}
