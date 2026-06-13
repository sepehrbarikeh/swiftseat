package handlers

import "swift-seat/internal/service"


type EventHandler struct {
	svc *service.EventService
}

func NewEventHandler(svc *service.EventService) *EventHandler {
	return &EventHandler{
		svc: svc,
	}
}