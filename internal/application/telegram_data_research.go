package application

import (
	"context"
	"fmt"
	"strconv"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/logger"
	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/reader"
	"github.com/go-faster/errors"
	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
)

type TelegramDataResearch struct {
	auth   *auth.Flow
	client *telegram.Client
	logger *logger.Logger
	gaps   *updates.Manager
}

func NewTelegramDataResearch() *TelegramDataResearch {
	// 1. Настройка логгера
	myLogger := logger.New()

	// 2. Инициализация системы обработки обновлений
	dispatcher := tg.NewUpdateDispatcher()
	gaps := updates.New(updates.Config{
		Handler: dispatcher,
		Logger:  myLogger.Named("updates"),
	})

	// 3. Настройка аутентификации
	authFlow := auth.NewFlow(
		examples.Terminal{},
		auth.SendCodeOptions{},
	)

	// 4. Хранилище сессии
	sessionStorage := domain.NewFileStorage("session.json")

	// 5. Чтение переменных окружения
	envReader := reader.NewEnvReader()
	apiIDstr, exists := envReader.GetEnv("TELEGRAM_API_ID")

	if !exists {
		myLogger.Fatal("TELEGRAM_API_ID not found")
	}

	apiID, isInt := strconv.Atoi(apiIDstr)
	if isInt != nil {
		myLogger.Fatal("TELEGRAM_API_ID is not integer")
	}

	apiHash, exists := envReader.GetEnv("TELEGRAM_API_HASH")
	if !exists {
		myLogger.Fatal("TELEGRAM_API_HASH not found")
	}

	// 6. Создание Telegram клиента
	client := telegram.NewClient(
		apiID,   // Ваш API ID из Telegram
		apiHash, // Ваш API Hash из Telegram
		telegram.Options{
			Logger:         myLogger.Named("client"),
			UpdateHandler:  gaps,
			SessionStorage: sessionStorage,
			Middlewares: []telegram.Middleware{
				hook.UpdateHook(gaps.Handle),
			},
		},
	)

	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		message, ok := u.Message.AsNotEmpty()
		if !ok {
			return nil
		}

		// Извлекаем Peer отправителя
		fromPeer, ok := message.GetFromID()
		if !ok {
			return nil
		}

		// Обрабатываем только сообщения от пользователей
		userPeer, ok := fromPeer.(*tg.PeerUser)
		if !ok {
			return nil
		}

		// Получаем информацию о пользователе
		var username string
		if user, exists := e.Users[userPeer.UserID]; exists {
			username = user.Username
			if username == "" {
				username = fmt.Sprintf("user%d", userPeer.UserID)
			}
		} else {
			// Если пользователя нет в сущностях, делаем запрос к API
			users, err := client.API().UsersGetUsers(ctx, []tg.InputUserClass{
				&tg.InputUser{UserID: userPeer.UserID},
			})

			if err == nil && len(users) > 0 {
				if fullUser, ok := users[0].(*tg.User); ok {
					username = fullUser.Username
					if username == "" {
						username = fmt.Sprintf("user%d", userPeer.UserID)
					}
					e.Users[userPeer.UserID] = fullUser // Кэшируем результат
				}
			}
		}

		myLogger.Logger.Info("New message",
			zap.String("text", message.(*tg.Message).GetMessage()),
			zap.String("sender", username),
			zap.Int64("id", userPeer.UserID),
		)
		return nil
	})

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		message, ok := u.Message.AsNotEmpty()
		if !ok {
			fmt.Println("a")
			return nil
		}

		channelPeer, ok := message.GetPeerID().(*tg.PeerChannel)
		if !ok {
			fmt.Println("aa")
			return nil
		}

		fmt.Println(channelPeer.ChannelID)

		var channelName string
		if channel, exists := e.Channels[channelPeer.ChannelID]; exists {
			channelName = channel.Title
			if channelName == "" {
				channelName = fmt.Sprintf("channel%d", channelPeer.ChannelID)
			}
		} else {
			// Если пользователя нет в сущностях, делаем запрос к API
			channels, err := client.API().ChannelsGetChannels(ctx, []tg.InputChannelClass{
				&tg.InputChannel{ChannelID: channelPeer.ChannelID},
			})

			if err == nil && len(channels.GetChats()) > 0 {
				if channel, ok := channels.GetChats()[0].(*tg.Channel); ok {
					channelName = channel.Title
					if channelName == "" {
						channelName = fmt.Sprintf("channel%d", channelPeer.ChannelID)
					}
					e.Channels[channelPeer.ChannelID] = channel // Кэшируем результат
				}
			}
		}

		if message.GetPost() {
			myLogger.Logger.Info("New post",
				zap.String("text", message.(*tg.Message).GetMessage()),
				zap.String("channel", channelName),
			)
		} else {
			fromPeer, ok := message.GetFromID()
			if !ok {
				fmt.Println("aaa")
				return nil
			} else {
				userPeer, ok := fromPeer.(*tg.PeerUser)
				if !ok {
					fmt.Println("aaaa")
					return nil
				}

				// Получаем информацию о пользователе
				var username string
				if user, exists := e.Users[userPeer.UserID]; exists {
					username = user.Username
					if username == "" {
						username = fmt.Sprintf("user%d", userPeer.UserID)
					}
				} else {
					// Если пользователя нет в сущностях, делаем запрос к API
					users, err := client.API().UsersGetUsers(ctx, []tg.InputUserClass{
						&tg.InputUser{UserID: userPeer.UserID},
					})

					if err == nil && len(users) > 0 {
						if fullUser, ok := users[0].(*tg.User); ok {
							username = fullUser.Username
							if username == "" {
								username = fmt.Sprintf("user%d", userPeer.UserID)
							}
							e.Users[userPeer.UserID] = fullUser // Кэшируем результат
						}
					}
				}
				myLogger.Logger.Info("New message",
					zap.String("text", message.(*tg.Message).GetMessage()),
					zap.String("sender", username),
					zap.String("channel", channelName),
				)
			}
		}

		return nil
	})

	return &TelegramDataResearch{
		client: client,
		auth:   &authFlow,
		logger: myLogger,
		gaps:   gaps,
	}
}

func (t *TelegramDataResearch) Run(ctx context.Context) error {
	defer func() { _ = t.logger.Sync() }()

	err := t.client.Run(ctx, t.research())

	return err
}

func (t *TelegramDataResearch) research() func(context.Context) error {
	return func(ctx context.Context) error {
		// 1. Аутентификация
		if err := t.client.Auth().IfNecessary(ctx, *t.auth); err != nil {
			return errors.Wrap(err, "auth")
		}

		user, err := t.client.Self(ctx)
		if err != nil {
			return errors.Wrap(err, "call self")
		}

		// Важно: активируем получение обновлений
		_, err = t.client.API().UpdatesGetState(ctx)
		if err != nil {
			return errors.Wrap(err, "get updates state")
		}

		return t.gaps.Run(ctx, t.client.API(), user.ID, updates.AuthOptions{
			OnStart: func(ctx context.Context) {
				t.logger.Info("Gaps started")
			},
		})
	}
}
