package websocket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingInterval   = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufferSize = 32
)

type InboundHandler func(client *Client, msg InboundMessage)

type Client struct {
	UserID uuid.UUID
	ChatID uuid.UUID

	ctx     context.Context
	conn    *websocket.Conn
	hub     *Hub
	send    chan OutboundMessage
	handler InboundHandler
}

func NewClient(ctx context.Context, hub *Hub, conn *websocket.Conn, userID, chatID uuid.UUID, handler InboundHandler) *Client {
	return &Client{
		UserID:  userID,
		ChatID:  chatID,
		ctx:     ctx,
		conn:    conn,
		hub:     hub,
		send:    make(chan OutboundMessage, sendBufferSize),
		handler: handler,
	}
}

func (c *Client) Enqueue(msg OutboundMessage) {
	select {
	case c.send <- msg:
	default:
		c.Close()
	}
}

func (c *Client) Close() {
	c.hub.Unsubscribe(c.ChatID, c)
	_ = c.conn.Close()
}

func (c *Client) Run() {
	go c.writePump()
	c.readPump()
}

func (c *Client) readPump() {
	defer c.Close()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg InboundMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			c.Enqueue(OutboundMessage{Type: OutboundTypeError, Error: "invalid message format"})
			continue
		}

		c.handler(c, msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
