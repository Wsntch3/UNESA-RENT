package objectstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrNotFound dipakai pemanggil untuk membedakan objek yang tidak ada dari error lain.
var ErrNotFound = errors.New("objek tidak ditemukan")

const requestTimeout = 30 * time.Second

// Supabase menyimpan objek lewat Supabase Storage REST API.
type Supabase struct {
	baseURL    string
	serviceKey string
	bucket     string
	http       *http.Client
}

func NewSupabase(baseURL, serviceKey, bucket string) *Supabase {
	return &Supabase{
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		serviceKey: strings.TrimSpace(serviceKey),
		bucket:     strings.TrimSpace(bucket),
		http: &http.Client{
			Timeout:       requestTimeout,
			CheckRedirect: noCrossHostRedirect,
		},
	}
}

// noCrossHostRedirect mengizinkan redirect hanya ke skema dan host asal.
func noCrossHostRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return errors.New("redirect Supabase tidak valid")
	}
	first := via[0].URL
	if req.URL.Scheme != first.Scheme || req.URL.Host != first.Host {
		return errors.New("redirect Supabase ke host lain ditolak")
	}
	if len(via) >= 10 {
		return errors.New("terlalu banyak redirect Supabase")
	}
	return nil
}

func (s *Supabase) Put(ctx context.Context, key string, data []byte, contentType string) error {
	u, err := s.objectURL(key)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("membuat request unggahan Supabase: %w", err)
	}
	s.authorize(req)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "false")

	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("mengunggah objek Supabase: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return statusError("mengunggah objek Supabase", resp.StatusCode)
	}
	return nil
}

func (s *Supabase) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	u, err := s.objectURL(key)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("membuat request unduhan Supabase: %w", err)
	}
	s.authorize(req)

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mengunduh objek Supabase: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, statusError("mengunduh objek Supabase", resp.StatusCode)
	}
	return resp.Body, nil
}

func (s *Supabase) Delete(ctx context.Context, key string) error {
	u, err := s.objectURL(key)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("membuat request hapus Supabase: %w", err)
	}
	s.authorize(req)

	resp, err := s.http.Do(req)
	if err != nil {
		return fmt.Errorf("menghapus objek Supabase: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return statusError("menghapus objek Supabase", resp.StatusCode)
	}
	return nil
}

func (s *Supabase) authorize(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
}

func (s *Supabase) objectURL(key string) (string, error) {
	if s.baseURL == "" || s.bucket == "" {
		return "", errors.New("konfigurasi Supabase Storage belum lengkap")
	}
	parts := strings.Split(key, "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("key objek tidak valid")
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return s.baseURL + "/storage/v1/object/" + url.PathEscape(s.bucket) + "/" + strings.Join(escaped, "/"), nil
}

// statusError sengaja tidak menyertakan key atau isi respons.
func statusError(operation string, status int) error {
	if status == http.StatusNotFound {
		return fmt.Errorf("%s: %w", operation, ErrNotFound)
	}
	return fmt.Errorf("%s: status HTTP %d", operation, status)
}
