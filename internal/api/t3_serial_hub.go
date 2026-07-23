package api

import (
	"sync"

	"github.com/gorilla/websocket"
)

type T3SerialHub struct {
	mu      sync.RWMutex
	current string
	clients map[*websocket.Conn]struct{}
}

func NewT3SerialHub() *T3SerialHub {
	return &T3SerialHub{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

func (h *T3SerialHub) Current() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.current
}

func (h *T3SerialHub) Subscribe(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	current := h.current
	h.mu.Unlock()

	_ = writeT3SerialMessage(conn, current)
}

func (h *T3SerialHub) Unsubscribe(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

func (h *T3SerialHub) Broadcast(serial string) {
	h.mu.Lock()
	h.current = serial
	conns := make([]*websocket.Conn, 0, len(h.clients))
	for conn := range h.clients {
		conns = append(conns, conn)
	}
	h.mu.Unlock()

	for _, conn := range conns {
		if err := writeT3SerialMessage(conn, serial); err != nil {
			h.drop(conn)
		}
	}
}

func (h *T3SerialHub) drop(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	_ = conn.Close()
}

func writeT3SerialMessage(conn *websocket.Conn, serial string) error {
	return conn.WriteJSON(map[string]string{"serial": serial})
}
