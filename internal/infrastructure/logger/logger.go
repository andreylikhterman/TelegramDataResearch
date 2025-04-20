package logger

import (
	"log"
	"os"

	"github.com/andreylikhterman/TelegramDataResearch/internal/infrastructure/db"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	Logger *zap.Logger
}

func New() *Logger {
	bd := db.Connect()
	_, err := bd.Exec(`
    INSERT INTO channels (id, title, type, subscribers_counter)
    VALUES ($1, $2, $3, $4)
`, "tg_channel_123", "Технологии и код", "p", 4820)

	if err != nil {
		log.Fatalf("Failed to insert channel: %v", err)
	}

	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("failed to open log file: " + err.Error())
	}

	// Настройка уровня логирования и формата
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.LevelKey = "level"
	encoderCfg.TimeKey = "time"

	// Создаем ядро для записи в файл
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(logFile),
		zapcore.InfoLevel,
	)

	// Создаем логгер
	logger := zap.New(fileCore, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	return &Logger{Logger: logger}
}

func (l *Logger) Named(name string) *Logger {
	return &Logger{Logger: l.Logger.Named(name)}
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
