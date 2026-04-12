package ws

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageID := chi.URLParam(r, "id")
		if pageID == "" {
			http.Error(w, "missing page id", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := &Client{
			pageID: pageID,
			hub:    hub,
			conn:   conn,
			send:   make(chan []byte, 64),
		}

		hub.Join(pageID, client)

		go client.WritePump()
		go client.ReadPump()
	}
}
