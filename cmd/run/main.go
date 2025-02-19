package main

import (
	"context"
	"github.com/andreylikhterman/TelegramDataResearch/internal/application/runresearch"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := runresearch.RunResearch(ctx); err != nil {
		panic(err)
	}
}
