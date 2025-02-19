package reader

import (
	"os"

	"github.com/joho/godotenv"
)

type EnvReader struct{}

func NewEnvReader() *EnvReader {
	return &EnvReader{}
}

func (e *EnvReader) GetEnv(key string) (string, bool) {
	err := godotenv.Load()
	if err != nil {
		return "", false
	}

	value := os.Getenv(key)

	return value, true
}
