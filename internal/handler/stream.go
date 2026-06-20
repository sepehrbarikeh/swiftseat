package handlers

import (
	"bufio"
	"fmt"
	"swift-seat/internal/sse"
	"time"

	"github.com/gofiber/fiber/v2"
)

type SSEHandler struct {
	hub *sse.Hub
}

func NewSSEHandler(hub *sse.Hub) *SSEHandler {
	return &SSEHandler{hub: hub}
}

func (h *SSEHandler) StreamEvents(c *fiber.Ctx) error {

    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")
    c.Set("Connection", "keep-alive")

    msgChan := make(chan []byte,10)

    h.hub.Register(msgChan)
    defer h.hub.Unregister(msgChan)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {

		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
	
		send := func(event string, data string) bool {
			_, err := fmt.Fprintf(
				w,
				"event: %s\ndata: %s\n\n",
				event,
				data,
			)
			if err != nil {
				return false
			}
			return w.Flush() == nil
		}
	
		if !send("connected", "{}") {
			return
		}
	
		for {
	
			select {
	
			case <-ticker.C:
				if !send("heartbeat", `"ping"`) {
					return
				}
	
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
	
				if !send("seat_update", string(msg)) {
					return
				}
			}
		}
	})

    return nil
}
