package httpapi

import "testing"

func TestTicketIsOneTime(t *testing.T) {
	store := newTicketStore()
	token, err := store.issue("user-1", "rental-1")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	ticket, ok := store.consume(token)
	if !ok {
		t.Fatal("first consume should succeed")
	}
	if ticket.UserID != "user-1" || ticket.RentalID != "rental-1" {
		t.Fatalf("unexpected ticket: %+v", ticket)
	}

	if _, ok := store.consume(token); ok {
		t.Fatal("second consume should fail")
	}
}
