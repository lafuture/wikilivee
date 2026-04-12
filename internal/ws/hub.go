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
}

type Message struct {
	Type    string         `json:"type"`
	UserID  string         `json:"userId"`
	Payload *UpdatePayload `json:"payload,omitempty"`
}

type UpdatePayload struct {
	Title   string         `json:"title"`
	Content []models.Block `json:"content"`
	Version int            `json:"version"`
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]bool
	db    PageRepo
}

func NewHub(db PageRepo) *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]bool),
		db:    db,
	}
}

func (h *Hub) Join(pageID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[pageID] == nil {
		h.rooms[pageID] = make(map[*Client]bool)
	}
	h.rooms[pageID][c] = true
}

func (h *Hub) Leave(pageID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms[pageID], c)
	if len(h.rooms[pageID]) == 0 {
		delete(h.rooms, pageID)
	}
}

func (h *Hub) Handle(ctx context.Context, pageID string, sender *Client, raw []byte) {
	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("ws: bad message from page %s: %v", pageID, err)
		return
	}

	if msg.Type == "update" && msg.Payload != nil {
		_, err := h.db.SavePage(ctx, pageID, msg.Payload.Title, msg.Payload.Content, msg.Payload.Version)
		if err != nil {
			log.Printf("ws: failed to save page %s: %v", pageID, err)
		}
	}

	h.broadcast(pageID, sender, raw)
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
