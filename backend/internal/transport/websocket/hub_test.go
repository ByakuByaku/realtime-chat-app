package websocket

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeSubscriber struct {
	received chan OutboundMessage
}

func newFakeSubscriber() *fakeSubscriber {
	return &fakeSubscriber{received: make(chan OutboundMessage, 1)}
}

func (f *fakeSubscriber) Enqueue(msg OutboundMessage) {
	f.received <- msg
}

func TestHubBroadcastDeliversToSubscriber(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	chatID := uuid.New()
	sub := newFakeSubscriber()
	hub.Subscribe(chatID, sub)

	hub.Broadcast(chatID, OutboundMessage{
		Type:    OutboundTypeMessage,
		Message: &MessagePayload{ChatID: chatID, Body: "hi"},
	})

	select {
	case got := <-sub.received:
		if got.Message.Body != "hi" {
			t.Fatalf("unexpected body: %q", got.Message.Body)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestHubUnsubscribeStopsDelivery(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	chatID := uuid.New()
	sub := newFakeSubscriber()
	hub.Subscribe(chatID, sub)
	hub.Unsubscribe(chatID, sub)

	hub.Broadcast(chatID, OutboundMessage{Type: OutboundTypeMessage})

	select {
	case <-sub.received:
		t.Fatal("expected no message after unsubscribe")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestHubBroadcastOnlyReachesSubscribedChat(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	chatA, chatB := uuid.New(), uuid.New()
	sub := newFakeSubscriber()
	hub.Subscribe(chatA, sub)

	hub.Broadcast(chatB, OutboundMessage{Type: OutboundTypeMessage})

	select {
	case <-sub.received:
		t.Fatal("received message for a chat it did not subscribe to")
	case <-time.After(200 * time.Millisecond):
	}
}
