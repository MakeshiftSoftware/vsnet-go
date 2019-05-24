package main

import (
	"github.com/gorilla/websocket"
)

// Hub implementation
type Hub struct {
	clients    map[string]*Client // Registered clients
	inbound    chan *Message      // Inbound messages channel
	register   chan *Client       // Register requests from clients
	unregister chan *Client       // Unregister requests from clients
}

// NewHub creates a new hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		inbound:    make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Start starts the hub
func (h *Hub) Start() {
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
}

// Stop stops the hub
func (h *Hub) Stop() {
	// @TODO implement
}

// OnClientConnected occurs when a new client connection is accepted
func (h *Hub) OnClientConnected(sock *websocket.Conn) {
	// @TODO validation (token / game ID / existing connection)
	c := NewClient("abcde", sock, h)
	h.register <- c
	c.Process()
}

// registerClient
func (h *Hub) registerClient(c *Client) {
	h.clients[c.id] = c
}

// unregisterClient
func (h *Hub) unregisterClient(c *Client) {
	c.sock.Close()
	delete(h.clients, c.id)
	close(c.send)
}

// relayMessage
func (h *Hub) relayMessage(msg *Message) {
	// @TODO
	// Instead of sending message to all clients,
	// iterate through recipient ids in parsed message
	// and send message data to those recipients
	for _, client := range h.clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(h.clients, client.id)
		}
	}
}
