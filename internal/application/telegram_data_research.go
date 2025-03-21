package application

import (
	"context"
	"fmt"
	"strconv"
	"sync"

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
	auths         []*auth.Flow
	clients       []*telegram.Client
	logger        *logger.Logger
	gaps          []*updates.Manager
	messages_chan *chan domain.Message
	posts_chan    *chan domain.Post
	mtx           sync.Mutex
}

func NewTelegramDataResearch() *TelegramDataResearch {
	// Инициализация логгера
	lg := logger.New()

	// Настройка обработчика обновлений

	// Хранилище сессии

	// Чтение API ID и Hash из переменных окружения
	ID, hash := getCredentials(reader.NewEnvReader(), lg)
	numOfAccounts := getNum(reader.NewEnvReader(), lg)
	ch_posts := make(chan domain.Post, 10)
	ch_messages := make(chan domain.Message, 10)
	var clients []*telegram.Client
	var authFlows []*auth.Flow
	var res_gaps []*updates.Manager
	for ind := 0; ind < numOfAccounts; ind++ {
		dispatcher := tg.NewUpdateDispatcher()
		gaps := updates.New(updates.Config{
			Handler: dispatcher,
			Logger:  lg.Named("updates" + strconv.Itoa(ind)).Logger,
		})

		// Настройка аутентификации через терминал
		authFlow := auth.NewFlow(examples.Terminal{}, auth.SendCodeOptions{})
		authFlows = append(authFlows, &authFlow)
		sessionStorage := domain.NewFileStorage("session" + strconv.Itoa(ind) + ".json")
		client := telegram.NewClient(ID, hash, telegram.Options{
			Logger:         lg.Named("client" + strconv.Itoa(ind)).Logger,
			UpdateHandler:  gaps,
			SessionStorage: sessionStorage,
			Middlewares:    []telegram.Middleware{hook.UpdateHook(gaps.Handle)},
		})
		clients = append(clients, client)
		res_gaps = append(res_gaps, gaps)
		registerHandlers(&dispatcher, client, ch_posts, ch_messages, lg)
	}
	// Создание Telegram-клиента

	// Регистрация обработчиков обновлений

	return &TelegramDataResearch{
		clients:       clients,
		auths:         authFlows,
		logger:        lg,
		gaps:          res_gaps,
		posts_chan:    &ch_posts,
		messages_chan: &ch_messages,
		mtx:           sync.Mutex{},
	}
}

func (t *TelegramDataResearch) Run(ctx context.Context) error {
	defer func() { _ = t.logger.Sync() }()
	var wg sync.WaitGroup
	var err error
	for numOfAccount, client := range t.clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = client.Run(ctx, t.research(numOfAccount))
		}()
	}
	go func() {
		for {
			fmt.Println("waiting for messages")
			for len(*t.messages_chan) < 10 {
			}
			fmt.Println("Читатель начал чтение 10 сообщений:")
			for i := 0; i < 10; i++ {
				msg := <-*t.messages_chan
				t.logger.Logger.Info("New message",
					zap.String("text", msg.Text),
					zap.Int("id", msg.Comment_id),
					zap.Int("post_id", msg.Post_id),
					zap.Int("user_id", msg.User_id),
					zap.String("sender", msg.User_name),
					zap.Int64("channel_id", msg.Channel_id),
					zap.String("channel", msg.Channel_name),
				)
			}
			fmt.Println("Чтение 10 сообщений завершено.")
		}
	}()
	go func() {
		for {
			fmt.Println("waiting for posts")
			for len(*t.posts_chan) < 10 {
			}

			fmt.Println("Читатель начал чтение 10 постов:")
			for i := 0; i < 10; i++ {
				msg := <-*t.posts_chan
				t.logger.Logger.Info("New post",
					zap.String("text", msg.Text),
					zap.Int("post_id", msg.Post_id),
					zap.Int64("channel_id", msg.Channel_id),
					zap.String("channel", msg.Channel),
				)
			}
			fmt.Println("Чтение 10 постов завершено.")
		}
	}()
	wg.Wait()
	close(*t.posts_chan)
	close(*t.messages_chan)
	return err
}

func (t *TelegramDataResearch) research(numOfAccount int) func(context.Context) error {
	return func(ctx context.Context) error {
		// Аутентификация, если необходимо
		t.mtx.Lock()
		if err := t.clients[numOfAccount].Auth().IfNecessary(ctx, *t.auths[numOfAccount]); err != nil {
			return errors.Wrap(err, "auth")
		}
		t.mtx.Unlock()
		// Получение информации о текущем пользователе
		user, err := t.clients[numOfAccount].Self(ctx)
		if err != nil {
			return errors.Wrap(err, "call self")
		}
		// Получение состояния обновлений
		if _, err = t.clients[numOfAccount].API().UpdatesGetState(ctx); err != nil {
			return errors.Wrap(err, "get updates state")
		}

		// Запуск менеджера обновлений
		return t.gaps[numOfAccount].Run(ctx, t.clients[numOfAccount].API(), user.ID, updates.AuthOptions{
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

func getNum(envReader *reader.EnvReader, lg *logger.Logger) int {
	num, exists := envReader.GetEnv("NUM_OF_ACCOUNTS")
	if !exists {
		lg.Fatal("NUM_OF_ACCOUNTS not found")
	}

	numInt, err := strconv.Atoi(num)
	if err != nil {
		lg.Fatal("NUM_OF_ACCOUNTS is not integer")
	}
	return numInt
}
