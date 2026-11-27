package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	wsTicketTTL      = 60 * time.Second
	wsWriteTimeout   = 5 * time.Second
	wsPingInterval   = 30 * time.Second
	wsMessagesPerMin = 30
)

type wsTicket struct {
	UserID   string
	RentalID string
	Expires  time.Time
}

type ticketStore struct {
	mu      sync.Mutex
	tickets map[string]wsTicket
}

func newTicketStore() *ticketStore {
	return &ticketStore{tickets: make(map[string]wsTicket)}
}

func (s *ticketStore) issue(userID, rentalID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	for key, t := range s.tickets {
		if now.After(t.Expires) {
			delete(s.tickets, key)
		}
	}
	s.tickets[token] = wsTicket{UserID: userID, RentalID: rentalID, Expires: now.Add(wsTicketTTL)}
	return token, nil
}

func (s *ticketStore) consume(token string) (wsTicket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[token]
	if !ok {
		return wsTicket{}, false
	}
	delete(s.tickets, token)
	if time.Now().After(t.Expires) {
		return wsTicket{}, false
	}
	return t, true
}

type chatClient struct {
	conn *websocket.Conn
	mu   sync.Mutex

	userID string

	sent   int
	window time.Time
}

func (c *chatClient) write(ctx context.Context, data []byte) error {
	ctx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
	defer cancel()

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *chatClient) ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
	defer cancel()

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Ping(ctx)
}

func (c *chatClient) allowMessage(now time.Time) bool {
	if now.After(c.window) {
		c.window = now.Add(time.Minute)
		c.sent = 0
	}
	c.sent++
	return c.sent <= wsMessagesPerMin
}

type chatHub struct {
	mu    sync.RWMutex
	rooms map[string]map[*chatClient]struct{}
}

func newChatHub() *chatHub {
	return &chatHub{rooms: make(map[string]map[*chatClient]struct{})}
}

func (h *chatHub) add(rentalID string, client *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[rentalID] == nil {
		h.rooms[rentalID] = make(map[*chatClient]struct{})
	}
	h.rooms[rentalID][client] = struct{}{}
}

func (h *chatHub) remove(rentalID string, client *chatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room := h.rooms[rentalID]; room != nil {
		delete(room, client)
		if len(room) == 0 {
			delete(h.rooms, rentalID)
		}
	}
}

func (h *chatHub) broadcast(ctx context.Context, rentalID string, data []byte) {
	h.mu.RLock()
	clients := make([]*chatClient, 0, len(h.rooms[rentalID]))
	for client := range h.rooms[rentalID] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if err := client.write(ctx, data); err != nil {
			_ = client.conn.CloseNow()
			h.remove(rentalID, client)
		}
	}
}
