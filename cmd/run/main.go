package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"runtime"

	"github.com/andreylikhterman/TelegramDataResearch/internal/application"
)

func main() {
	app := application.NewTelegramDataResearch()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	//go monitorMemory()
	if err := app.Run(ctx); err != nil {
		panic(err)
	}
}

func monitorMemory() {
	var m runtime.MemStats
	for {
		runtime.ReadMemStats(&m)
		fmt.Printf("Heap in use: %d MB\n", m.HeapInuse/1024/1024)
		fmt.Printf("Total allocated in heap: %d MB\n", m.TotalAlloc/1024/1024)
		fmt.Printf("Currently allocated for whole app: %d MB\n", m.Sys/1024/1024)
		fmt.Printf("Heap objects: %d\n", m.HeapObjects)
		fmt.Printf("Stack in use: %d bytes\n", m.StackInuse)
		fmt.Printf("Stack allocated: %d bytes\n", m.StackSys)
		fmt.Printf("Garbage Collector runs: %d\n", m.NumGC)
		time.Sleep(1000 * time.Millisecond) // Частота замера
	}
}
