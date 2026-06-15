package sse

import (
	"fmt"
	"sync"
)

type Hub struct {
	clients map[chan []byte]bool
	mu      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{clients: make(map[chan []byte]bool)}
}

func (h *Hub) Register(c chan []byte) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *Hub) Unregister(c chan []byte) {
	h.mu.Lock()
	delete(h.clients, c)
	close(c) // حتما کانال رو ببند
	h.mu.Unlock()
}

func (h *Hub) Broadcast(msg []byte) {
	h.mu.Lock()
	fmt.Printf("📢 Broadcasting: %s\n", string(msg)) // این لاگ رو اضافه کن
	for c := range h.clients {
		c <- msg
	}
	h.mu.Unlock()
}