package RunResearch

import (
	"context"
	"time"

	"github.com/andreylikhterman/TelegramDataResearch/internal/application/PrintChannels"
	"github.com/andreylikhterman/TelegramDataResearch/internal/domain/Filestorage"
	"github.com/andreylikhterman/TelegramDataResearch/internal/domain/PublicChannel"
	"github.com/go-faster/errors"
	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	updhook "github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func RunResearch(ctx context.Context) error {
	log, _ := zap.NewDevelopment(
		zap.IncreaseLevel(zapcore.InfoLevel),
		zap.AddStacktrace(zapcore.FatalLevel),
	)
	defer func() { _ = log.Sync() }()

	d := tg.NewUpdateDispatcher()
	gaps := updates.New(updates.Config{
		Handler: d,
		Logger:  log.Named("gaps"),
	})

	flow := auth.NewFlow(examples.Terminal{}, auth.SendCodeOptions{})
	fileStorage := Filestorage.NewFileStorage("session.json")

	client := telegram.NewClient(
		20981738,                           // Ваш api_id
		"a60a5eea86f42605f459a51c6e393cc4", // Ваш api_hash
		telegram.Options{
			Logger:         log,
			UpdateHandler:  gaps,
			SessionStorage: fileStorage,
			Middlewares: []telegram.Middleware{
				updhook.UpdateHook(gaps.Handle),
			},
		},
	)

	return client.Run(ctx, func(ctx context.Context) error {
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return errors.Wrap(err, "auth")
		}

		user, err := client.Self(ctx)
		if err != nil {
			return errors.Wrap(err, "call self")
		}

		// Список username каналов
		channelUsernames := []string{"anillarionov", "cherevatstreams"}

		var channels []PublicChannel.PublicChannel
		for _, username := range channelUsernames {
			res, err := client.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
				Username: username,
			})
			if err != nil {
				log.Error("Не удалось разрешить username", zap.String("username", username), zap.Error(err))
				continue
			}
			var channel *tg.Channel
			for _, chat := range res.Chats {
				if ch, ok := chat.(*tg.Channel); ok {
					channel = ch
					break
				}
			}
			if channel == nil {
				log.Error("Канал не найден", zap.String("username", username))
				continue
			}
			channels = append(channels, PublicChannel.PublicChannel{
				ID:         channel.ID,
				AccessHash: channel.AccessHash,
				Title:      channel.Title,
			})
		}

		if len(channels) == 0 {
			return errors.New("нет разрешенных каналов")
		}

		// Начальный запрос для всех каналов.
		for _, ch := range channels {
			PrintChannels.PrintChannels(ctx, client, log, ch)
		}

		// Периодически (каждую минуту) обновляем историю постов и комментариев для каждого канала.
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					for _, ch := range channels {
						PrintChannels.PrintChannels(ctx, client, log, ch)
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		// Остальная логика обработки обновлений
		_, err = client.API().UpdatesGetState(ctx)
		if err != nil {
			return errors.Wrap(err, "get updates state")
		}

		return gaps.Run(ctx, client.API(), user.ID, updates.AuthOptions{
			OnStart: func(ctx context.Context) {
				log.Info("Gaps started")
			},
		})
	})
}
