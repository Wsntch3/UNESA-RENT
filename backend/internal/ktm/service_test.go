package ktm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

type fakeStore struct {
	state    UserState
	stateErr error

	createID  string
	createErr error
	created   []NewSubmission

	submission Submission
	getErr     error
	list       []Submission

	verifyErr error
	rejectErr error
	revokeErr error

	actions []string
}

func (f *fakeStore) Create(_ context.Context, in NewSubmission) (string, error) {
	f.created = append(f.created, in)
	if f.createErr != nil {
		return "", f.createErr
	}
	return f.createID, nil
}

func (f *fakeStore) GetByID(_ context.Context, _ string) (Submission, error) {
	if f.getErr != nil {
		return Submission{}, f.getErr
	}
	return f.submission, nil
}

func (f *fakeStore) ListByUser(_ context.Context, _ string, _ int) ([]Submission, error) {
	return f.list, nil
}

func (f *fakeStore) ListByStatus(_ context.Context, _ string, _, _ int) ([]Submission, error) {
	return f.list, nil
}

func (f *fakeStore) UserState(_ context.Context, _ string) (UserState, error) {
	return f.state, f.stateErr
}

func (f *fakeStore) MarkVerified(_ context.Context, _, _ string) error {
	return f.verifyErr
}

func (f *fakeStore) MarkRejected(_ context.Context, _, _, _ string) error {
	return f.rejectErr
}

func (f *fakeStore) MarkRevoked(_ context.Context, _, _, _ string) error {
	return f.revokeErr
}

func (f *fakeStore) LogAction(_ context.Context, _, action, _, _ string) error {
	f.actions = append(f.actions, action)
	return nil
}

type fakeStorage struct {
	saved    map[string][]byte
	saveErr  error
	deleted  []string
	openData []byte
	openErr  error
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{saved: make(map[string][]byte)}
}

func (f *fakeStorage) Save(_ context.Context, key string, data []byte, _ string) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved[key] = append([]byte(nil), data...)
	return nil
}

func (f *fakeStorage) Open(_ context.Context, _ string) (io.ReadCloser, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	return io.NopCloser(bytes.NewReader(f.openData)), nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	delete(f.saved, key)
	f.deleted = append(f.deleted, key)
	return nil
}

// PNG signature cukup untuk http.DetectContentType.
var pngBytes = []byte("\x89PNG\r\n\x1a\n")

func eligibleState() UserState {
	return UserState{
		ID:     testUserID,
		Name:   "Mahasiswa Uji",
		Email:  "mhs@example.com",
		NIM:    "24051204001",
		Role:   roleMahasiswa,
		Status: statusActive,
	}
}

func TestSubmitRejectsUnsupportedType(t *testing.T) {
	svc := NewService(&fakeStore{}, newFakeStorage())

	_, err := svc.Submit(context.Background(), testUserID, []byte("bukan gambar"))
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("Submit() error = %v, want ErrUnsupportedType", err)
	}
}

func TestSubmitRejectsTooLarge(t *testing.T) {
	svc := NewService(&fakeStore{}, newFakeStorage())

	data := make([]byte, MaxUploadBytes+1)
	copy(data, pngBytes)

	_, err := svc.Submit(context.Background(), testUserID, data)
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("Submit() error = %v, want ErrFileTooLarge", err)
	}
}

func TestSubmitRejectsIneligibleUser(t *testing.T) {
	store := &fakeStore{state: UserState{ID: testUserID, Role: roleAdminDev, Status: statusActive}}
	svc := NewService(store, newFakeStorage())

	_, err := svc.Submit(context.Background(), testUserID, pngBytes)
	if !errors.Is(err, ErrIneligible) {
		t.Fatalf("Submit() error = %v, want ErrIneligible", err)
	}
}

func TestSubmitRejectsAlreadyVerified(t *testing.T) {
	state := eligibleState()
	state.Verified = true
	svc := NewService(&fakeStore{state: state}, newFakeStorage())

	_, err := svc.Submit(context.Background(), testUserID, pngBytes)
	if !errors.Is(err, ErrAlreadyVerified) {
		t.Fatalf("Submit() error = %v, want ErrAlreadyVerified", err)
	}
}

func TestSubmitSavesFileAndCreatesRow(t *testing.T) {
	state := eligibleState()
	store := &fakeStore{
		state:    state,
		createID: "sub-1",
		submission: Submission{
			ID:     "sub-1",
			UserID: testUserID,
			Status: StatusPending,
		},
	}
	files := newFakeStorage()
	svc := NewService(store, files)

	sub, err := svc.Submit(context.Background(), testUserID, pngBytes)
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if sub.ID != "sub-1" {
		t.Fatalf("submission ID = %q, want sub-1", sub.ID)
	}

	if len(store.created) != 1 {
		t.Fatalf("jumlah Create = %d, want 1", len(store.created))
	}
	in := store.created[0]
	if !storageKeyPattern.MatchString(in.StorageKey) {
		t.Fatalf("storage key = %q tidak sesuai pola", in.StorageKey)
	}
	if in.MimeType != "image/png" {
		t.Fatalf("mime = %q, want image/png", in.MimeType)
	}
	if in.FileSizeBytes != int64(len(pngBytes)) {
		t.Fatalf("size = %d, want %d", in.FileSizeBytes, len(pngBytes))
	}
	if len(in.ChecksumSHA256) != 64 {
		t.Fatalf("checksum length = %d, want 64", len(in.ChecksumSHA256))
	}
	if _, ok := files.saved[in.StorageKey]; !ok {
		t.Fatalf("file tidak tersimpan pada key %q", in.StorageKey)
	}
}

func TestSubmitDeletesFileWhenInsertFails(t *testing.T) {
	store := &fakeStore{
		state:     eligibleState(),
		createErr: ErrPendingExists,
	}
	files := newFakeStorage()
	svc := NewService(store, files)

	_, err := svc.Submit(context.Background(), testUserID, pngBytes)
	if !errors.Is(err, ErrPendingExists) {
		t.Fatalf("Submit() error = %v, want ErrPendingExists", err)
	}
	if len(files.deleted) != 1 {
		t.Fatalf("file yang dihapus = %d, want 1", len(files.deleted))
	}
}

func TestVerifyRequiresNIMConfirmation(t *testing.T) {
	svc := NewService(&fakeStore{}, newFakeStorage())

	_, err := svc.Verify(context.Background(), "admin-1", "sub-1", false)
	if !errors.Is(err, ErrNIMNotConfirmed) {
		t.Fatalf("Verify() error = %v, want ErrNIMNotConfirmed", err)
	}
}

func TestVerifyRejectsNonPending(t *testing.T) {
	store := &fakeStore{submission: Submission{ID: "sub-1", Status: StatusVerified}}
	svc := NewService(store, newFakeStorage())

	_, err := svc.Verify(context.Background(), "admin-1", "sub-1", true)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Verify() error = %v, want ErrInvalidTransition", err)
	}
}

func TestRejectRequiresReason(t *testing.T) {
	svc := NewService(&fakeStore{}, newFakeStorage())

	_, err := svc.Reject(context.Background(), "admin-1", "sub-1", "   ")
	if !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("Reject() error = %v, want ErrReasonRequired", err)
	}
}

func TestRevokeRejectsNonVerified(t *testing.T) {
	store := &fakeStore{submission: Submission{ID: "sub-1", Status: StatusPending}}
	svc := NewService(store, newFakeStorage())

	_, err := svc.Revoke(context.Background(), "admin-1", "sub-1", "KTM tidak valid")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Revoke() error = %v, want ErrInvalidTransition", err)
	}
}

func TestFileOnlyOwnerOrAdmin(t *testing.T) {
	store := &fakeStore{
		submission: Submission{
			ID:         "sub-1",
			UserID:     testUserID,
			StorageKey: "ktm/" + testUserID + "/0123456789abcdef0123456789abcdef.png",
			MimeType:   "image/png",
		},
	}
	files := newFakeStorage()
	files.openData = pngBytes
	svc := NewService(store, files)

	if _, _, err := svc.File(context.Background(), "99999999-9999-4999-8999-999999999999", roleMahasiswa, "sub-1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("File() pemilik lain error = %v, want ErrForbidden", err)
	}

	sub, rc, err := svc.File(context.Background(), testUserID, roleMahasiswa, "sub-1")
	if err != nil {
		t.Fatalf("File() pemilik error = %v", err)
	}
	rc.Close()
	if sub.ID != "sub-1" {
		t.Fatalf("submission ID = %q, want sub-1", sub.ID)
	}

	if _, adminRC, err := svc.File(context.Background(), "admin-1", roleAdminDev, "sub-1"); err != nil {
		t.Fatalf("File() admin error = %v", err)
	} else {
		adminRC.Close()
	}
}
