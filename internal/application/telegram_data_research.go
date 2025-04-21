package application

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/db"
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
	db            *db.MyDB
}

func NewTelegramDataResearch() *TelegramDataResearch {
	// Инициализация логгера
	lg := logger.New()
	db := db.MyDB{DB: db.Connect()}

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
		db:            &db,
	}
}

func (t *TelegramDataResearch) Run(ctx context.Context) error {
	defer func() { _ = t.logger.Sync() }()
	var wg sync.WaitGroup
	var err error
	channels := []string{"technodeus2023", "cherevatstreams", "shot_shot", "moscowach", "pravdadirty",
		"topor", "nexta_live", "Ateobreaking", "ru2ch", "moscow",
		"moscowachplus", "rt_russian", "RhymesMorgen", "smi_rf_moskva", "lentachtrue",
		"kosti", "petrovtel", "mashmoyka", "milinfolive", "live_piter",
		"nemorgenshtern", "chp_crimea", "kazancity", "spbtoday", "chtddd",
		"Petya_perviy", "oldlentach", "e1_news", "krd_tipich_ru", "region116_kazan",
		"svodka25", "tsargradtv", "piterach", "SuperRu", "kazan",
		"tvrain", "kursk_tipich", "chp_sochi", "chp_kavkaz", "rosich_russia"}
	for numOfAccount, client := range t.clients {
		wg.Add(1)
		go func(number int) {
			defer wg.Done()
			err = client.Run(ctx, t.research(number, channels))
		}(numOfAccount)
	}

	go func() {
		placeholders := make([]string, 10)
		for {
			for len(*t.messages_chan) < 10 {
			}
			args := make([]any, 0)
			for i := 0; i < 10; i++ {
				placeholders[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", i*7+1, i*7+2, i*7+3, i*7+4, i*7+5, i*7+6, i*7+7)
				msg := <-*t.messages_chan
				args = append(args, msg.Text, msg.Comment_id, msg.Post_id, msg.User_id,
					msg.User_name, msg.Channel_name, msg.Channel_id)
				go func(id int64, name string) {
					t.db.InsertUser(id, name)
				}(int64(msg.User_id), msg.User_name)

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
			t.db.InsertMessage(placeholders, args)
		}
	}()
	go func() {
		placeholders := make([]string, 10)

		for {
			for len(*t.posts_chan) < 10 {
			}
			args := make([]any, 0)
			for i := 0; i < 10; i++ {
				placeholders[i] = fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4)
				msg := <-*t.posts_chan
				args = append(args, msg.Text, msg.Post_id, msg.Channel_id, msg.Channel)
				t.logger.Logger.Info("New post",
					zap.String("text", msg.Text),
					zap.Int("post_id", msg.Post_id),
					zap.Int64("channel_id", msg.Channel_id),
					zap.String("channel", msg.Channel),
				)
			}
			t.db.InsertPost(placeholders, args)
		}
	}()
	wg.Wait()
	close(*t.posts_chan)
	close(*t.messages_chan)
	return err
}

func (t *TelegramDataResearch) research(numOfAccount int, Channels []string) func(context.Context) error {
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
		how_many_chats := len(Channels) / len(t.clients)
		fmt.Println(how_many_chats)
		if how_many_chats > 500 {
			how_many_chats = 500
		}
		channels, err := FetchChannelDataByNames(ctx, t.clients[numOfAccount], Channels[(how_many_chats*numOfAccount):(how_many_chats*(numOfAccount+1))])
		if err != nil {
			t.logger.Error("Ошибка получения данных о каналах " + "error" + err.Error())
			return err
		}
		for _, ch := range channels {
			// Приводим DiscussionPeer к *tg.InputPeerChannel
			peer, ok := ch.DiscussionPeer.(*tg.InputPeerChannel)
			if !ok {
				t.logger.Info("Пропускаем канал: нет привязанного чата " + "title" + ch.Title)
				time.Sleep(10 * time.Second)
				continue
			}

			// Проверяем, подписан ли пользователь уже на чат
			_, err := t.clients[numOfAccount].API().ChannelsGetParticipant(ctx, &tg.ChannelsGetParticipantRequest{

				Channel: &tg.InputChannel{
					ChannelID:  peer.ChannelID,
					AccessHash: peer.AccessHash,
				},
				Participant: &tg.InputPeerSelf{},
			})
			t.db.InsertChannel(ch.ID, ch.Title)
			if err == nil {
				t.logger.Info("Уже подписан на title " + ch.Title)
				time.Sleep(10 * time.Second)
				continue
			}

			// Подписываемся на чат, если еще не подписаны
			err = SubscribeToDiscussionChats(ctx, t.clients[numOfAccount], []domain.PublicChannel{ch})

			if err != nil {
				t.logger.Error("Ошибка при подписке на чат " + "title" + ch.Title + err.Error())
			} else {
				t.logger.Info("Успешно подписались на чат " + "title" + ch.Title)
			}
			time.Sleep(10 * time.Second)
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
