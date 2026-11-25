package rental

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestValidateRentalDates(t *testing.T) {
	today := todayWIB()
	day := func(n int) string { return today.AddDate(0, 0, n).Format(DateFormat) }

	tests := []struct {
		name    string
		in      DateInput
		wantErr bool
	}{
		{"same day", DateInput{StartDate: day(1), EndDate: day(1)}, false},
		{"max 30 days", DateInput{StartDate: day(1), EndDate: day(30)}, false},
		{"31 days", DateInput{StartDate: day(1), EndDate: day(31)}, true},
		{"past start", DateInput{StartDate: day(-1), EndDate: day(1)}, true},
		{"end before start", DateInput{StartDate: day(2), EndDate: day(1)}, true},
		{"bad format", DateInput{StartDate: "25-11-2026", EndDate: day(1)}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := validateRentalDates(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestApproveOnlyOwner(t *testing.T) {
	store := &fakeStore{rental: Rental{
		ID:      "r1",
		Owner:   Party{ID: "owner"},
		Renter:  Party{ID: "renter"},
		Status:  StatusPending,
		EndDate: todayWIB().AddDate(0, 0, 1).Format(DateFormat),
	}}
	svc := NewService(store)

	if _, err := svc.Approve(context.Background(), "renter", "r1"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("renter approve err = %v, want ErrForbidden", err)
	}

	got, err := svc.Approve(context.Background(), "owner", "r1")
	if err != nil {
		t.Fatalf("owner approve: %v", err)
	}
	if got.Status != StatusApproved {
		t.Fatalf("status = %s, want approved", got.Status)
	}
	if store.status != StatusApproved {
		t.Fatalf("store status = %s, want approved", store.status)
	}
}

func TestSendMessageRejectsClosedRental(t *testing.T) {
	store := &fakeStore{rental: Rental{
		ID:     "r1",
		Renter: Party{ID: "renter"},
		Owner:  Party{ID: "owner"},
		Status: StatusCompleted,
	}}
	svc := NewService(store)

	_, err := svc.SendMessage(context.Background(), "renter", "r1", "halo")
	if !errors.Is(err, ErrChatClosed) {
		t.Fatalf("err = %v, want ErrChatClosed", err)
	}
}

type fakeStore struct {
	Store
	rental Rental
	status Status
}

func (f *fakeStore) Get(ctx context.Context, id string) (Rental, error) {
	return f.rental, nil
}

func (f *fakeStore) UpdateStatus(ctx context.Context, id string, to Status, reason *string) error {
	f.status = to
	f.rental.Status = to
	return nil
}

func (f *fakeStore) AddEvent(ctx context.Context, rentalID string, from *Status, to Status, actorID string, note *string) error {
	return nil
}

func (f *fakeStore) CreateNotification(ctx context.Context, userID, kind, title, body string, data map[string]any) error {
	return nil
}

func (f *fakeStore) Events(ctx context.Context, rentalID string) ([]Event, error) {
	return nil, nil
}

func (f *fakeStore) SaveMessage(ctx context.Context, rentalID, senderID, body string) (Message, error) {
	return Message{ID: "m1", RentalID: rentalID, Body: body, CreatedAt: time.Now()}, nil
}
