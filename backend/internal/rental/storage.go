package rental

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var (
	documentKeyPattern = regexp.MustCompile(`^rentals/[0-9a-f-]{36}/[0-9a-f]{32}\.(pdf|jpg|png)$`)
	ErrFileNotFound    = errors.New("file dokumen tidak ditemukan")
)

type Storage interface {
	Save(ctx context.Context, key string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type LocalStorage struct {
	dir string
}

func NewLocalStorage(dir string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return &LocalStorage{dir: dir}, nil
}

func (s *LocalStorage) Save(_ context.Context, key string, data []byte) error {
	if !documentKeyPattern.MatchString(key) {
		return ErrFileNotFound
	}
	path := filepath.Join(s.dir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if !documentKeyPattern.MatchString(key) {
		return nil, ErrFileNotFound
	}
	file, err := os.Open(filepath.Join(s.dir, filepath.FromSlash(key)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrFileNotFound
	}
	return file, err
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	if !documentKeyPattern.MatchString(key) {
		return ErrFileNotFound
	}
	err := os.Remove(filepath.Join(s.dir, filepath.FromSlash(key)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func newDocumentKey(rentalID, ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "rentals/" + rentalID + "/" + hex.EncodeToString(b) + ext, nil
}
