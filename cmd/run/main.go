package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/andreylikhterman/TelegramDataResearch/internal/application/runresearch"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := runresearch.RunResearch(ctx); err != nil {
		panic(err)
	}
}
