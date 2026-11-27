package catalog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Storage interface {
	Save(ctx context.Context, key string, data []byte, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

var photoKeyPattern = regexp.MustCompile(`^listings/[0-9a-f-]{36}/[0-9a-f]{32}\.(jpg|png)$`)

var ErrInvalidStorageKey = errors.New("storage key foto tidak valid")

type LocalStorage struct {
	root string
}

func NewLocalStorage(root string) (*LocalStorage, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("direktori penyimpanan foto listing kosong")
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve direktori foto listing: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("membuat direktori foto listing: %w", err)
	}
	return &LocalStorage{root: abs}, nil
}

func (s *LocalStorage) Save(ctx context.Context, key string, data []byte, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path, err := s.resolve(key)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("membuat direktori foto listing: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".photo-*.tmp")
	if err != nil {
		return fmt.Errorf("membuat file sementara foto: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("mengatur permission foto: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("menulis foto: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("menutup foto: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("menyimpan foto: %w", err)
	}
	return nil
}

func (s *LocalStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path, err := s.resolve(key)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("membuka foto: %w", err)
	}
	return f, nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path, err := s.resolve(key)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("menghapus foto: %w", err)
	}
	return nil
}

func (s *LocalStorage) resolve(key string) (string, error) {
	if !photoKeyPattern.MatchString(key) {
		return "", ErrInvalidStorageKey
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}
