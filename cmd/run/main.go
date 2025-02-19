package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/andreylikhterman/TelegramDataResearch/internal/application"
)

func main() {
	app := application.NewTelegramDataResearch()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		panic(err)
	}
}
