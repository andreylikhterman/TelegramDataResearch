package reader

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/andreylikhterman/TelegramDataResearch/internal/domain"
	"github.com/joho/godotenv"
)

type EnvReader struct{}

func NewEnvReader() *EnvReader {
	return &EnvReader{}
}

func (e *EnvReader) GetEnv(key string) (string, bool) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
		return "", false
	}

	value := os.Getenv(key)

	return value, true
}

func (e *EnvReader) GetChannels() ([]domain.Account, bool) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
		return nil, false
	}
	channelsJSON := os.Getenv("ACCOUNTS")
	var channels []domain.Account
	err = json.Unmarshal([]byte(channelsJSON), &channels)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, false
	}
	return channels, true

}
