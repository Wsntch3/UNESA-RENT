package rental

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"unesarent/backend/internal/objectstore"
)

type SupabaseStorage struct {
	client *objectstore.Supabase
}

func NewSupabaseStorage(client *objectstore.Supabase) *SupabaseStorage {
	return &SupabaseStorage{client: client}
}

func (s *SupabaseStorage) Save(ctx context.Context, key string, data []byte) error {
	if !documentKeyPattern.MatchString(key) {
		return ErrFileNotFound
	}
	return s.client.Put(ctx, key, data, contentTypeForKey(key))
}

func (s *SupabaseStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !documentKeyPattern.MatchString(key) {
		return nil, ErrFileNotFound
	}
	file, err := s.client.Get(ctx, key)
	if errors.Is(err, objectstore.ErrNotFound) {
		return nil, ErrFileNotFound
	}
	return file, err
}

func (s *SupabaseStorage) Delete(ctx context.Context, key string) error {
	if !documentKeyPattern.MatchString(key) {
		return ErrFileNotFound
	}
	return s.client.Delete(ctx, key)
}

func contentTypeForKey(key string) string {
	switch strings.ToLower(filepath.Ext(key)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "application/pdf"
	}
}
