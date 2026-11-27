package ktm

import (
	"context"
	"errors"
	"fmt"
	"io"

	"unesarent/backend/internal/objectstore"
)

type SupabaseStorage struct {
	client *objectstore.Supabase
}

func NewSupabaseStorage(client *objectstore.Supabase) *SupabaseStorage {
	return &SupabaseStorage{client: client}
}

func (s *SupabaseStorage) Save(ctx context.Context, key string, data []byte, contentType string) error {
	if err := s.validateKey(key); err != nil {
		return err
	}
	if err := s.client.Put(ctx, key, data, contentType); err != nil {
		return fmt.Errorf("menyimpan file KTM: %w", err)
	}
	return nil
}

func (s *SupabaseStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := s.validateKey(key); err != nil {
		return nil, err
	}
	r, err := s.client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, objectstore.ErrNotFound) {
			return nil, ErrFilePurged
		}
		return nil, fmt.Errorf("membuka file KTM: %w", err)
	}
	return r, nil
}

func (s *SupabaseStorage) Delete(ctx context.Context, key string) error {
	if err := s.validateKey(key); err != nil {
		return err
	}
	if err := s.client.Delete(ctx, key); err != nil {
		if errors.Is(err, objectstore.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("menghapus file KTM: %w", err)
	}
	return nil
}

func (s *SupabaseStorage) validateKey(key string) error {
	if s.client == nil {
		return errors.New("client Supabase Storage kosong")
	}
	if !storageKeyPattern.MatchString(key) {
		return ErrInvalidStorageKey
	}
	return nil
}
