package application

import (
	"context"
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
)

type TelegramDataResearch struct {
	auth   *auth.Flow
	client *telegram.Client
	logger *logger.Logger
	gaps   *updates.Manager
}

func NewTelegramDataResearch() *TelegramDataResearch {
	// Инициализация логгера
	lg := logger.New()

	// Настройка обработчика обновлений
	dispatcher := tg.NewUpdateDispatcher()
	gaps := updates.New(updates.Config{
		Handler: dispatcher,
		Logger:  lg.Named("updates").Logger,
	})

	// Настройка аутентификации через терминал
	authFlow := auth.NewFlow(examples.Terminal{}, auth.SendCodeOptions{})

	// Хранилище сессии
	sessionStorage := domain.NewFileStorage("session.json")

	// Чтение API ID и Hash из переменных окружения
	apiID, apiHash := getCredentials(reader.NewEnvReader(), lg)

	// Создание Telegram-клиента
	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		Logger:         lg.Named("client").Logger,
		UpdateHandler:  gaps,
		SessionStorage: sessionStorage,
		Middlewares:    []telegram.Middleware{hook.UpdateHook(gaps.Handle)},
	})

	// Регистрация обработчиков обновлений
	registerHandlers(&dispatcher, client, lg)

	return &TelegramDataResearch{
		client: client,
		auth:   &authFlow,
		logger: lg,
		gaps:   gaps,
	}
}

func (t *TelegramDataResearch) Run(ctx context.Context) error {
	defer func() { _ = t.logger.Sync() }()
	return t.client.Run(ctx, t.research())
}

func (t *TelegramDataResearch) research() func(context.Context) error {
	return func(ctx context.Context) error {
		// Аутентификация, если необходимо
		if err := t.client.Auth().IfNecessary(ctx, *t.auth); err != nil {
			return errors.Wrap(err, "auth")
		}

		// Получение информации о текущем пользователе
		user, err := t.client.Self(ctx)
		if err != nil {
			return errors.Wrap(err, "call self")
		}

		// Получение состояния обновлений
		if _, err = t.client.API().UpdatesGetState(ctx); err != nil {
			return errors.Wrap(err, "get updates state")
		}

		// Запуск менеджера обновлений
		return t.gaps.Run(ctx, t.client.API(), user.ID, updates.AuthOptions{
			OnStart: func(ctx context.Context) {
				t.logger.Info("Gaps started")
			},
		})
	}
}

func getCredentials(envReader *reader.EnvReader, lg *logger.Logger) (int, string) {
	apiIDStr, exists := envReader.GetEnv("TELEGRAM_API_ID")
	if !exists {
		lg.Fatal("TELEGRAM_API_ID not found")
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		lg.Fatal("TELEGRAM_API_ID is not integer")
	}

	apiHash, exists := envReader.GetEnv("TELEGRAM_API_HASH")
	if !exists {
		lg.Fatal("TELEGRAM_API_HASH not found")
	}

	return apiID, apiHash
}
