package logger

import (
	"os"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	Logger *zap.Logger
}

func New() *Logger {
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic("failed to open log file: " + err.Error())
	}

	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	core := zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), zapcore.InfoLevel)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
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
