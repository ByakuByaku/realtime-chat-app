package websocket

import (
	"sync"

	"github.com/google/uuid"
)

type Subscriber interface {
	Enqueue(OutboundMessage)
}

type subscription struct {
	chatID uuid.UUID
	client Subscriber
}

type broadcastMessage struct {
	chatID  uuid.UUID
	payload OutboundMessage
}

type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[Subscriber]struct{}

	register   chan subscription
	unregister chan subscription
	broadcast  chan broadcastMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[Subscriber]struct{}),
		register:   make(chan subscription),
		unregister: make(chan subscription),
		broadcast:  make(chan broadcastMessage, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case sub := <-h.register:
			h.addSubscriber(sub)
		case sub := <-h.unregister:
			h.removeSubscriber(sub)
		case msg := <-h.broadcast:
			h.dispatch(msg)
		}
	}
}

func (h *Hub) addSubscriber(sub subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[sub.chatID] == nil {
		h.clients[sub.chatID] = make(map[Subscriber]struct{})
	}
	h.clients[sub.chatID][sub.client] = struct{}{}
}

func (h *Hub) removeSubscriber(sub subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients[sub.chatID], sub.client)
	if len(h.clients[sub.chatID]) == 0 {
		delete(h.clients, sub.chatID)
	}
}

func (h *Hub) dispatch(msg broadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[msg.chatID] {
		client.Enqueue(msg.payload)
	}
}

func (h *Hub) Subscribe(chatID uuid.UUID, client Subscriber) {
	h.register <- subscription{chatID: chatID, client: client}
}

func (h *Hub) Unsubscribe(chatID uuid.UUID, client Subscriber) {
	h.unregister <- subscription{chatID: chatID, client: client}
}

func (h *Hub) Broadcast(chatID uuid.UUID, payload OutboundMessage) {
	h.broadcast <- broadcastMessage{chatID: chatID, payload: payload}
}
