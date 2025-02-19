package runresearch

import (
	"context"
	"github.com/andreylikhterman/TelegramDataResearch/internal/domain/filestorage"
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
	fileStorage := filestorage.NewFileStorage("session.json")

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

	return client.Run(ctx, runClientLogic(client, &flow, log, gaps))
}
