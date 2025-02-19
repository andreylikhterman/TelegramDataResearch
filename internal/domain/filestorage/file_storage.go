package filestorage

import (
	"context"
	"os"

	"github.com/gotd/td/session"
)

type FileStorage struct {
	filename string
}

func NewFileStorage(filename string) *FileStorage {
	return &FileStorage{filename: filename}
}

func (fs *FileStorage) LoadSession(ctx context.Context) ([]byte, error) {
	data, err := os.ReadFile(fs.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, session.ErrNotFound
		}

		return nil, err
	}

	return data, nil
}

func (fs *FileStorage) StoreSession(ctx context.Context, data []byte) error {
	return os.WriteFile(fs.filename, data, 0o600)
}
