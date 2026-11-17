package ktm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"
)

const testUserID = "11111111-1111-4111-8111-111111111111"

func TestLocalStorageRoundTrip(t *testing.T) {
	st, err := NewLocalStorage(filepath.Join(t.TempDir(), "ktm"))
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	ctx := context.Background()
	key := "ktm/" + testUserID + "/0123456789abcdef0123456789abcdef.png"
	want := []byte("isi file ktm")

	if err := st.Save(ctx, key, want, "image/png"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	rc, err := st.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("isi file = %q, want %q", got, want)
	}

	if err := st.Delete(ctx, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := st.Open(ctx, key); !errors.Is(err, ErrFilePurged) {
		t.Fatalf("Open() setelah Delete error = %v, want ErrFilePurged", err)
	}
}

func TestLocalStorageRejectsUnsafeKeys(t *testing.T) {
	st, err := NewLocalStorage(filepath.Join(t.TempDir(), "ktm"))
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	keys := []string{
		"../escape.png",
		"/absolute.png",
		"ktm/../escape.png",
		"ktm/not-a-uuid/abcdef.png",
		"ktm/" + testUserID + "/../../escape.png",
		"ktm/" + testUserID + "/0123456789abcdef0123456789abcdef.exe",
	}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			if err := st.Save(context.Background(), key, []byte("x"), "image/png"); !errors.Is(err, ErrInvalidStorageKey) {
				t.Fatalf("Save(%q) error = %v, want ErrInvalidStorageKey", key, err)
			}
		})
	}
}
