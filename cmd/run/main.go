package main

import (
	"context"
	"github.com/andreylikhterman/TelegramDataResearch/internal/application/RunResearch"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := RunResearch.RunResearch(ctx); err != nil {
		panic(err)
	}
}
