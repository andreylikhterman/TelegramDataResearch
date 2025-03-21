package application

import (
	"context"
	"fmt"

	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/logger"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

func registerHandlers(dispatcher *tg.UpdateDispatcher, client *telegram.Client, lg *logger.Logger) {
	dispatcher.OnNewMessage(handleNewMessage(client, lg))
	dispatcher.OnNewChannelMessage(handleNewChannelMessage(client, lg))
}

func handleNewMessage(client *telegram.Client, lg *logger.Logger) func(context.Context, tg.Entities, *tg.UpdateNewMessage) error {
	return func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		message, ok := u.Message.AsNotEmpty()
		if !ok {
			return nil
		}

		userID, username := GetUser(ctx, message, e, client)
		if userID == 0 {
			return nil
		}

		lg.Logger.Info("New message",
			zap.String("text", message.(*tg.Message).GetMessage()),
			zap.String("sender", username),
			zap.Int("id", userID),
		)
		return nil
	}
}

func handleNewChannelMessage(client *telegram.Client, lg *logger.Logger) func(context.Context, tg.Entities, *tg.UpdateNewChannelMessage) error {
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
			lg.Logger.Info("New post",
				zap.String("text", comment),
				zap.Int("post_id", commentID),
				zap.Int64("channel_id", channelPeer.ChannelID),
				zap.String("channel", channelName),
			)
		} else {
			lg.Logger.Info("New message",
				zap.String("text", comment),
				zap.Int("id", commentID),
				zap.Int("post_id", postID),
				zap.Int("user_id", userID),
				zap.String("sender", username),
				zap.Int64("channel_id", channelPeer.ChannelID),
				zap.String("channel", channelName),
			)
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
			&tg.InputUser{UserID: userPeer.UserID},
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
