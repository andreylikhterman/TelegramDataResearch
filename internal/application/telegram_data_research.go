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

var channels []string = []string{"technodeus2023", "cherevatstreams", "shot_shot", "moscowach", "pravdadirty",
	"topor", "nexta_live", "Ateobreaking", "ru2ch", "moscow",
	"moscowachplus", "rt_russian", "RhymesMorgen", "smi_rf_moskva", "lentachtrue",
	"kosti", "petrovtel", "mashmoyka", "milinfolive", "live_piter",
	"nemorgenshtern", "chp_crimea", "kazancity", "spbtoday", "chtddd",
	"Petya_perviy", "oldlentach", "e1_news", "krd_tipich_ru", "region116_kazan",
	"svodka25", "tsargradtv", "piterach", "SuperRu", "kazan",
	"tvrain", "kursk_tipich", "chp_sochi", "chp_kavkaz", "rosich_russia"}

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
	lg := logger.New()
	dbConn := db.MyDB{DB: db.Connect()}

	apiID, apiHash := getCredentials(reader.NewEnvReader(), lg)
	numAccounts := getNum(reader.NewEnvReader(), lg)

	chPosts := make(chan domain.Post, 1)
	chMessages := make(chan domain.Message, 10)

	clients, authFlows, gaps := initClients(numAccounts, apiID, apiHash, chPosts, chMessages, lg)

	return &TelegramDataResearch{
		clients:       clients,
		auths:         authFlows,
		logger:        lg,
		gaps:          gaps,
		posts_chan:    &chPosts,
		messages_chan: &chMessages,
		mtx:           sync.Mutex{},
		db:            &dbConn,
	}
}

func initClients(numAccounts, apiID int, apiHash string, chPosts chan domain.Post, chMessages chan domain.Message, lg *logger.Logger) ([]*telegram.Client, []*auth.Flow, []*updates.Manager) {
	var clients []*telegram.Client
	var authFlows []*auth.Flow
	var gapsList []*updates.Manager

	for i := range numAccounts {
		dispatcher := tg.NewUpdateDispatcher()
		gaps := updates.New(updates.Config{
			Handler: dispatcher,
			Logger:  lg.Named("updates" + strconv.Itoa(i)).Logger,
		})

		authFlow := auth.NewFlow(examples.Terminal{}, auth.SendCodeOptions{})
		authFlows = append(authFlows, &authFlow)

		sessionStorage := domain.NewFileStorage(fmt.Sprintf("session%d.json", i))

		client := telegram.NewClient(apiID, apiHash, telegram.Options{
			Logger:         lg.Named("client" + strconv.Itoa(i)).Logger,
			UpdateHandler:  gaps,
			SessionStorage: sessionStorage,
			Middlewares:    []telegram.Middleware{hook.UpdateHook(gaps.Handle)},
		})

		registerHandlers(&dispatcher, client, chPosts, chMessages)

		clients = append(clients, client)
		gapsList = append(gapsList, gaps)
	}
	return clients, authFlows, gapsList
}

func (t *TelegramDataResearch) Run(ctx context.Context) error {
	t.db.InsertZeroPostIfNotExists()
	defer t.logger.Sync()

	channels := getChannelsList()

	var wg sync.WaitGroup
	var runErr error

	for i, client := range t.clients {
		wg.Add(1)
		go func(idx int, c *telegram.Client) {
			defer wg.Done()
			if err := c.Run(ctx, t.research(idx, channels)); err != nil {
				t.logger.Error("client run error", zap.Int("client", idx), zap.Error(err))
				runErr = err
			}
		}(i, client)
	}

	go t.handleMessages()
	go t.handlePosts()

	wg.Wait()
	close(*t.posts_chan)
	close(*t.messages_chan)

	return runErr
}

func (t *TelegramDataResearch) handleMessages() {
	wg := &sync.WaitGroup{}
	for {
		if len(*t.messages_chan) < 10 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var args []any
		for range 10 {
			msg := <-*t.messages_chan
			args = append(args, msg.Text, msg.CommentId, msg.PostId, msg.UserId, msg.UserName, msg.ChannelName, msg.ChannelId, msg.Data, msg.RepliedTo)

			wg.Add(1)
			go func(id int64, name string) {
				defer wg.Done()
				t.db.InsertUser(id, name)
			}(int64(msg.UserId), msg.UserName)

			t.logger.Info("New message",
				zap.String("text", msg.Text),
				zap.Int("comment_id", msg.CommentId),
				zap.Int("post_id", msg.PostId),
				zap.Int("user_id", msg.UserId),
				zap.String("sender", msg.UserName),
				zap.Int64("channel_id", msg.ChannelId),
				zap.String("channel", msg.ChannelName),
			)
		}
		wg.Wait()
		t.db.InsertMessage(args)
	}
}

func (t *TelegramDataResearch) handlePosts() {
	placeholders := make([]string, 1)
	for {
		if len(*t.posts_chan) < 1 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var args []any
		for i := range 1 {
			placeholders[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", i*5+1, i*5+2, i*5+3, i*5+4, i*5+5)
			post := <-*t.posts_chan
			args = append(args, post.Text, post.PostId, post.ChannelId, post.ChannelName, post.Data)

			t.logger.Info("New post",
				zap.String("text", post.Text),
				zap.Int("post_id", post.PostId),
				zap.Int64("channel_id", post.ChannelId),
				zap.String("channel", post.ChannelName),
			)
		}
		t.db.InsertPost(placeholders, args)
	}
}

func (t *TelegramDataResearch) research(accountIdx int, allChannels []string) func(context.Context) error {
	return func(ctx context.Context) error {
		t.mtx.Lock()
		if err := t.clients[accountIdx].Auth().IfNecessary(ctx, *t.auths[accountIdx]); err != nil {
			t.mtx.Unlock()
			return errors.Wrap(err, "auth failed")
		}
		t.mtx.Unlock()

		user, err := t.clients[accountIdx].Self(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to get self info")
		}

		if _, err = t.clients[accountIdx].API().UpdatesGetState(ctx); err != nil {
			return errors.Wrap(err, "failed to get updates state")
		}

		chunks := chunkChannels(allChannels, len(t.clients))
		channels := chunks[accountIdx]

		fetchedChannels, err := FetchChannelDataByNames(ctx, t.clients[accountIdx], channels)
		if err != nil {
			t.logger.Error("Failed to fetch channel data", zap.Error(err))
			return err
		}

		for _, ch := range fetchedChannels {
			if err := t.processChannel(ctx, accountIdx, ch); err != nil {
				t.logger.Error("Failed to process channel", zap.String("title", ch.Title), zap.Error(err))
			}
			time.Sleep(5 * time.Second)
		}

		return t.gaps[accountIdx].Run(ctx, t.clients[accountIdx].API(), user.ID, updates.AuthOptions{
			OnStart: func(context.Context) {
				t.logger.Info("Gaps started", zap.Int("account", accountIdx))
			},
		})
	}
}

func (t *TelegramDataResearch) processChannel(ctx context.Context, accountIdx int, ch domain.PublicChannel) error {
	peer, ok := ch.DiscussionPeer.(*tg.InputPeerChannel)
	if !ok {
		t.logger.Info("Skipping channel without discussion peer", zap.String("title", ch.Title))
		return nil
	}

	_, err := t.clients[accountIdx].API().ChannelsGetParticipant(ctx, &tg.ChannelsGetParticipantRequest{
		Channel: &tg.InputChannel{
			ChannelID:  peer.ChannelID,
			AccessHash: peer.AccessHash,
		},
		Participant: &tg.InputPeerSelf{},
	})

	t.db.InsertChannel(peer.ChannelID, ch.Title, int(ch.SubsCount), ch.Type)

	if err == nil {
		t.logger.Info("Already subscribed", zap.String("title", ch.Title))
		latestMessageID, err := getLastMessageID(ctx, t.clients[accountIdx], ch)
		if err != nil {
			t.logger.Error("Failed to get last message ID", zap.String("title", ch.Title), zap.Error(err))
			return err
		}

		lastStoredID, err := t.db.GetMaxCommentIDByChannel(peer.ChannelID)
		if err != nil {
			t.logger.Error("Failed to get last stored message ID", zap.String("title", ch.Title), zap.Error(err))
			lastStoredID = 0
		}
		if latestMessageID > int(lastStoredID) && lastStoredID != 0 {
			entities := tg.Entities{Channels: make(map[int64]*tg.Channel), Users: make(map[int64]*tg.User)}
			err = GetHistory(ch, lastStoredID, int64(latestMessageID), *t.posts_chan, *t.messages_chan, t.db, t.clients[accountIdx], entities)
			if err != nil {
				t.logger.Error("Failed to fetch history", zap.String("title", ch.Title), zap.Error(err))
				return err
			}
		}
		return nil
	}

	err = SubscribeToDiscussionChats(ctx, t.clients[accountIdx], []domain.PublicChannel{ch}, t.logger)
	if err != nil {
		t.logger.Error("Failed to subscribe", zap.String("title", ch.Title), zap.Error(err))
		return err
	}

	t.logger.Info("Subscribed to chat", zap.String("title", ch.Title))
	return nil
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
	numStr, exists := envReader.GetEnv("NUM_OF_ACCOUNTS")
	if !exists {
		lg.Fatal("NUM_OF_ACCOUNTS not found")
	}
	numInt, err := strconv.Atoi(numStr)
	if err != nil {
		lg.Fatal("NUM_OF_ACCOUNTS is not integer")
	}
	return numInt
}

func getChannelsList() []string {
	return channels
}

func getLastMessageID(ctx context.Context, client *telegram.Client, ch domain.PublicChannel) (int, error) {
	history, err := client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  ch.DiscussionPeer,
		Limit: 1,
	})
	if err != nil {
		return 0, err
	}

	switch h := history.(type) {
	case *tg.MessagesChannelMessages:
		if len(h.Messages) == 0 {
			return 0, nil
		}
		if msg, ok := h.Messages[0].(*tg.Message); ok {
			return msg.ID, nil
		}
	}
	return 0, nil
}

func chunkChannels(channels []string, parts int) [][]string {
	chunkSize := (len(channels) + parts - 1) / parts
	var chunks [][]string
	for i := 0; i < len(channels); i += chunkSize {
		end := i + chunkSize
		if end > len(channels) {
			end = len(channels)
		}
		chunks = append(chunks, channels[i:end])
	}
	return chunks
}
