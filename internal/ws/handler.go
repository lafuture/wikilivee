package ws

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

type CursorPayload struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var presenceColors = []string{
	"#3b82f6", "#ef4444", "#10b981", "#f59e0b",
	"#8b5cf6", "#ec4899", "#06b6d4", "#84cc16",
}

func randomColor() string {
	return presenceColors[rand.Intn(len(presenceColors))]
}

func NewHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageID := chi.URLParam(r, "id")
		if pageID == "" {
			http.Error(w, "missing page id", http.StatusBadRequest)
			return
		}

		r.Proto = "HTTP/1.1"
		r.ProtoMajor = 1
		r.ProtoMinor = 1

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		userID := r.URL.Query().Get("userId")
		name := r.URL.Query().Get("name")
		color := r.URL.Query().Get("color")

		if userID == "" {
			userID = fmt.Sprintf("anon-%d", rand.Intn(99999))
		}
		if name == "" {
			name = "Anonymous"
		}
		if color == "" {
			color = randomColor()
		}

		client := &Client{
			pageID: pageID,
			userID: userID,
			hub:    hub,
			conn:   conn,
			send:   make(chan []byte, 64),
		}

		hub.Join(pageID, client, PresenceUser{
			UserID: userID,
			Name:   name,
			Color:  color,
		})

		go client.WritePump()
		go client.ReadPump()
	}
}
