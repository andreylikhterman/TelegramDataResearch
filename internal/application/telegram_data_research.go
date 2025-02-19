package application

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/logger"
	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/output"
	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/reader"
	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
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
			return fmt.Errorf("failed to auth: %w", err)
		}

		// 2. Получение информации о пользователе
		user, err := t.client.Self(ctx)
		if err != nil {
			return fmt.Errorf("failed to get self: %w", err)
		}

		// 3. Список каналов
		channelNames := []string{"anillarionov", "cherevatstreams"}

		var channels []domain.PublicChannel

		// 4. Получение информации о каналах
		for _, name := range channelNames {
			res, err := t.client.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
				Username: name,
			})

			if err != nil {
				t.logger.Error(fmt.Sprintf("Failed to find name of channel: %s", name))
				continue
			}

			channel := res.Chats[0].(*tg.Channel)

			if channel == nil {
				t.logger.Error(fmt.Sprintf("Channel not found: %s", name))
				continue
			}

			channels = append(channels, *domain.NewPublicChannel(channel.ID, channel.AccessHash, channel.Title))
		}

		if len(channels) == 0 {
			t.logger.Error("No allowed channels")
		}

		// 5. Вывод информации о каналах
		for _, ch := range channels {
			output.PrintChannels(ctx, t.client, t.logger.Logger, ch)
		}

		// 6. Периодическое обновление информации о каналах
		go t.updateChannel(ctx, channels)

		// 7. Получение обновлений
		return t.gaps.Run(ctx, t.client.API(), user.ID, updates.AuthOptions{
			OnStart: func(_ context.Context) {
				t.logger.Info("Gaps started")
			},
		})
	}
}

func (t *TelegramDataResearch) updateChannel(ctx context.Context, channels []domain.PublicChannel) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, ch := range channels {
				output.PrintChannels(ctx, t.client, t.logger.Logger, ch)
			}
		case <-ctx.Done():
			return
		}
	}
}
