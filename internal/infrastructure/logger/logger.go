package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	Logger *zap.Logger
}

func New() *Logger {
	logger, _ := zap.NewDevelopment(
		zap.IncreaseLevel(zapcore.InfoLevel),  // Уровень логирования Info и выше
		zap.AddStacktrace(zapcore.FatalLevel), // Вывод стека только для Fatal
	)

	return &Logger{
		Logger: logger,
	}
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

func (l *Logger) Named(name string) *zap.Logger {
	return l.Logger.Named(name)
}

func (l *Logger) Fatal(msg string) {
	l.Logger.Fatal(msg)
}

func (l *Logger) Error(msg string) {
	l.Logger.Error(msg)
}

func (l *Logger) Info(msg string) {
	l.Logger.Info(msg)
}
