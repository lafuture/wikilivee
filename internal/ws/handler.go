package ws

import (
	"math/rand"
	"net/http"
	"wikilivee/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

type CursorPayload struct {
	Anchor int `json:"anchor"`
	Head   int `json:"head"`
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
		userID, _ := r.Context().Value(middleware.UserIDKey).(string)
		username, _ := r.Context().Value(middleware.UsernameKey).(string)
		if userID == "" || username == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		canEdit, err := hub.CanUserEditPage(r.Context(), pageID, userID, username)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		if !canEdit {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		r.Proto = "HTTP/1.1"
		r.ProtoMajor = 1
		r.ProtoMinor = 1

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		color := r.URL.Query().Get("color")
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
			Name:   username,
			Color:  color,
		})

		go client.WritePump()
		go client.ReadPump()
	}
}
