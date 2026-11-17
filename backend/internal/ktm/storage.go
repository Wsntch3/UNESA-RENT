package ktm

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

// Storage adalah abstraksi penyimpanan file KTM.
//
// Saat ini hanya ada implementasi lokal (LocalStorage). Untuk Supabase nanti,
// cukup buat implementasi Storage baru yang memakai Supabase Storage API/SDK;
// service dan handler tidak perlu berubah.
type Storage interface {
	Save(ctx context.Context, key string, data []byte, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// storageKeyPattern membatasi key ke format yang dibuat service:
// ktm/<uuid-user>/<32 hex>.(jpg|png)
var storageKeyPattern = regexp.MustCompile(`^ktm/[0-9a-f-]{36}/[0-9a-f]{32}\.(jpg|png)$`)

var ErrInvalidStorageKey = errors.New("storage key KTM tidak valid")

// LocalStorage menyimpan file di filesystem lokal dengan permission 0600.
type LocalStorage struct {
	root string
}

func NewLocalStorage(root string) (*LocalStorage, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("direktori penyimpanan KTM kosong")
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve direktori KTM: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("membuat direktori KTM: %w", err)
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
		return fmt.Errorf("membuat direktori file KTM: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".ktm-*.tmp")
	if err != nil {
		return fmt.Errorf("membuat file sementara KTM: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("mengatur permission file KTM: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("menulis file KTM: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("menutup file KTM: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("menyimpan file KTM: %w", err)
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
			return nil, ErrFilePurged
		}
		return nil, fmt.Errorf("membuka file KTM: %w", err)
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
		return fmt.Errorf("menghapus file KTM: %w", err)
	}
	return nil
}

func (s *LocalStorage) resolve(key string) (string, error) {
	if !storageKeyPattern.MatchString(key) {
		return "", ErrInvalidStorageKey
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}
