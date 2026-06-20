package sse

import (
	"sync"
)

type Hub struct {
	clients map[chan []byte]bool
	mu      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[chan []byte]bool),
	}
}

func (h *Hub) Register(c chan []byte) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *Hub) Unregister(c chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c)
	}
}

func (h *Hub) Broadcast(msg []byte) {

    h.mu.Lock()
    defer h.mu.Unlock()

    for c := range h.clients {

        select {
        case c <- msg:
        default:
            close(c)
            delete(h.clients, c)
        }
    }
}