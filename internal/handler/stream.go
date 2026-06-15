package handlers

import (
	"bufio"
	"fmt"
	"swift-seat/internal/sse"

	"github.com/gofiber/fiber/v2"
)

type SSEHandler struct{
	hub *sse.Hub
}

func NewSSEHandler(hub *sse.Hub) *SSEHandler {
	return &SSEHandler{
		hub: hub,
	}
}

// در فایل handler (مثلاً internal/handlers/sse.go)
func (h *SSEHandler) StreamEvents(c *fiber.Ctx) error {
    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")
    c.Set("Connection", "keep-alive")
    c.Set("Transfer-Encoding", "chunked")

    msgChan := make(chan []byte)
    h.hub.Register(msgChan)
    defer h.hub.Unregister(msgChan)

    // استفاده از NotifyClose برای اینکه وقتی یوزر صفحه رو بست، دیسکانکت بشه
    notify := c.Context().Done()

    c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
        for {
            select {
            case <-notify:
                return
            case msg := <-msgChan:
                fmt.Fprintf(w, "data: %s\n\n", msg)
                w.Flush()
            }
        }
    })
    return nil
}