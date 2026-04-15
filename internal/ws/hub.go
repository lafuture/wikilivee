package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"wikilivee/internal/models"
)

type PageRepo interface {
	SavePage(ctx context.Context, id, title string, content []models.Block, version int) (int, error)
	CanUserEditPage(ctx context.Context, pageID, userID, username string) (bool, error)
}

type Message struct {
	Type    string         `json:"type"`
	UserID  string         `json:"userId"`
	Payload *UpdatePayload `json:"payload,omitempty"`
	Cursor  *CursorPayload `json:"cursor,omitempty"`
}

type PresenceUser struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Color  string `json:"color"`
}

type UpdatePayload struct {
	Title   string         `json:"title"`
	Content []models.Block `json:"content"`
	Version int            `json:"version"`
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]bool
	users map[*Client]PresenceUser
	db    PageRepo
}

func NewHub(db PageRepo) *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]bool),
		users: make(map[*Client]PresenceUser),
		db:    db,
	}
}

func (h *Hub) Join(pageID string, c *Client, user PresenceUser) {
	h.mu.Lock()
	if h.rooms[pageID] == nil {
		h.rooms[pageID] = make(map[*Client]bool)
	}
	h.users[c] = user
	h.rooms[pageID][c] = true
	h.mu.Unlock()

	h.broadcastPresence(pageID)
}

func (h *Hub) Leave(pageID string, c *Client) {
	h.mu.Lock()
	delete(h.rooms[pageID], c)
	delete(h.users, c)
	if len(h.rooms[pageID]) == 0 {
		delete(h.rooms, pageID)
	}
	h.mu.Unlock()

	h.broadcastPresence(pageID)
}

func (h *Hub) Handle(ctx context.Context, pageID string, sender *Client, raw []byte) {
	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("ws: bad message from page %s: %v", pageID, err)
		return
	}

	switch msg.Type {

	case "update":
		if msg.Payload != nil {
			h.db.SavePage(ctx, pageID, msg.Payload.Title, msg.Payload.Content, msg.Payload.Version)
		}
		h.broadcast(pageID, sender, raw)
	case "cursor":
		h.broadcast(pageID, sender, raw)
	}
}

func (h *Hub) CanUserEditPage(ctx context.Context, pageID, userID, username string) (bool, error) {
	return h.db.CanUserEditPage(ctx, pageID, userID, username)
}

func (h *Hub) broadcast(pageID string, sender *Client, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[pageID] {
		if c != sender {
			select {
			case c.send <- msg:
			default:
			}
		}
	}
}

func (h *Hub) broadcastPresence(pageID string) {
	h.mu.RLock()
	var users []PresenceUser
	for c := range h.rooms[pageID] {
		users = append(users, h.users[c])
	}
	clients := make([]*Client, 0, len(h.rooms[pageID]))
	for c := range h.rooms[pageID] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	msg, err := json.Marshal(map[string]any{
		"type":  "presence",
		"users": users,
	})
	if err != nil {
		return
	}

	for _, c := range clients {
		select {
		case c.send <- msg:
		default:
		}
	}
}
