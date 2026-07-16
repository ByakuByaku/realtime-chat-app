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
	userID uuid.UUID
	client Subscriber
}

type broadcastMessage struct {
	chatID  uuid.UUID
	payload OutboundMessage
}

type disconnectRequest struct {
	chatID uuid.UUID
	userID uuid.UUID
}

type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[Subscriber]uuid.UUID

	register   chan subscription
	unregister chan subscription
	broadcast  chan broadcastMessage
	disconnect chan disconnectRequest
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[Subscriber]uuid.UUID),
		register:   make(chan subscription),
		unregister: make(chan subscription),
		broadcast:  make(chan broadcastMessage, 256),
		disconnect: make(chan disconnectRequest),
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
		case req := <-h.disconnect:
			h.disconnectUser(req)
		}
	}
}

func (h *Hub) addSubscriber(sub subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[sub.chatID] == nil {
		h.clients[sub.chatID] = make(map[Subscriber]uuid.UUID)
	}
	h.clients[sub.chatID][sub.client] = sub.userID
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

func (h *Hub) disconnectUser(req disconnectRequest) {
	h.mu.Lock()
	var targets []Subscriber
	for client, userID := range h.clients[req.chatID] {
		if userID == req.userID {
			targets = append(targets, client)
		}
	}
	for _, client := range targets {
		delete(h.clients[req.chatID], client)
	}
	if len(h.clients[req.chatID]) == 0 {
		delete(h.clients, req.chatID)
	}
	h.mu.Unlock()

	for _, client := range targets {
		if closer, ok := client.(interface{ Close() }); ok {
			go closer.Close()
		}
	}
}

func (h *Hub) Subscribe(chatID, userID uuid.UUID, client Subscriber) {
	h.register <- subscription{chatID: chatID, userID: userID, client: client}
}

func (h *Hub) Unsubscribe(chatID uuid.UUID, client Subscriber) {
	h.unregister <- subscription{chatID: chatID, client: client}
}

func (h *Hub) Broadcast(chatID uuid.UUID, payload OutboundMessage) {
	h.broadcast <- broadcastMessage{chatID: chatID, payload: payload}
}

func (h *Hub) DisconnectUser(chatID, userID uuid.UUID) {
	h.disconnect <- disconnectRequest{chatID: chatID, userID: userID}
}
