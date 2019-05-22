package main

import (
	"log"
)

// Hub implementation
type Hub struct {
	clients    map[string]*Client // Registered clients
	broadcast  chan []byte        // Inbound messages from clients
	register   chan *Client       // Register requests from clients
	unregister chan *Client       // Unregister requests from clients
}

func newHub() *Hub {
	hub := &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	go hub.run()

	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			if len(h.clients) >= cfg.maxConnections {
				log.Printf("[Warn] Max connections limit reached, dropping new connection")

				client.ws.Close()
			} else {
				log.Printf("[Info] Registering new client with id %v", client.getID())

				h.clients[client.getID()] = client
			}
		case client := <-h.unregister:
			if _, ok := h.clients[client.getID()]; ok {
				log.Printf("[Info] Unregistering client with id %v", client.getID())

				delete(h.clients, client.getID())
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
