package messenger

import (
	"github.com/gorilla/websocket"
)

// Hub implementation
type Hub struct {
	clients    map[string]*Client
	inbound    chan *Message
	register   chan *Client
	unregister chan *Client
	queue      *MessageQueue
}

// NewHub creates a new hub
func NewHub(name string, redisAddr string) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		inbound:    make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		queue:      NewMessageQueue(name, redisAddr),
	}
}

// Start starts the hub
func (h *Hub) Start() error {
	if err := h.queue.Start(); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case client := <-h.register:
				h.registerClient(client)
			case client := <-h.unregister:
				h.unregisterClient(client)
			case msg := <-h.inbound:
				h.relayMessage(msg)
			}
		}
	}()
	return nil
}

// Stop stops the hub
func (h *Hub) Stop() {
	h.queue.Stop()
}

// Ready checks if hub is ready
func (h *Hub) Ready() error {
	return h.queue.Ready()
}

// OnClientConnected occurs when a new client connection is accepted
func (h *Hub) OnClientConnected(sock *websocket.Conn) {
	c := NewClient("abcde", sock, h)
	h.register <- c
	c.Process()
}

// registerClient registers client to hub
func (h *Hub) registerClient(c *Client) {
	h.clients[c.id] = c
}

// unregisterClient unregisters client from hub
func (h *Hub) unregisterClient(c *Client) {
	delete(h.clients, c.id)
}

// relayMessage relays message to hub
func (h *Hub) relayMessage(msg *Message) {
	for _, client := range h.clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(h.clients, client.id)
		}
	}
}
