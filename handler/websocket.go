package handler

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"sync"
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins; consider restricting in production
	},
}

type WebSocketHub struct {
	clients   map[*websocket.Conn]bool
	broadcast chan []byte
	lock      sync.Mutex
}

var Hub = &WebSocketHub{
	clients:   make(map[*websocket.Conn]bool),
	broadcast: make(chan []byte),
}

func (h *WebSocketHub) Run() {
	for {
		message := <-h.broadcast
		h.lock.Lock()
		for client := range h.clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				client.Close()
				delete(h.clients, client)
			}
		}
		h.lock.Unlock()
	}
}

func WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	Hub.lock.Lock()
	Hub.clients[conn] = true
	Hub.lock.Unlock()
}
