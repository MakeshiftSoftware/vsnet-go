package messenger

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
)

// Hub implementation
type Hub struct {
	clients    map[ClientID]*Client
	inbound    chan *Message
	queue      chan *Message
	register   chan *Client
	unregister chan *Client
	broker     *Broker
	presence   *Presence
}

// NewHub creates new hub
func NewHub(name string, brokerAddr string, presenceAddr string) *Hub {
	h := &Hub{
		clients:    make(map[ClientID]*Client),
		inbound:    make(chan *Message),
		queue:      make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		presence:   NewPresence(name, presenceAddr),
	}

	h.broker = NewBroker(name, brokerAddr, h)
	return h
}

// Start starts hub
func (h *Hub) Start() error {
	log.Print("[info] starting hub...")

	if err := h.broker.Start(); err != nil {
		return err
	}

	if err := h.presence.Start(); err != nil {
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
				h.broadcast(msg)
			case msg := <-h.queue:
				h.send(msg)
			}
		}
	}()

	log.Print("[info] hub started")
	return nil
}

// Stop stops hub
func (h *Hub) Stop() (result error) {
	if err := h.broker.Stop(); err != nil {
		result = multierror.Append(result, err)
	}

	ids := make([]ClientID, len(h.clients))
	i := 0

	for id, client := range h.clients {
		close(client.outbound)
		client.sock.Close()
		ids[i] = id
		i++
	}

	if err := h.presence.RemoveMulti(ids); err != nil {
		result = multierror.Append(result, err)
	}

	if err := h.presence.Stop(); err != nil {
		result = multierror.Append(result, err)
	}

	return
}

// Ready checks if hub is ready
func (h *Hub) Ready() (result error) {
	if err := h.broker.Ready(); err != nil {
		result = multierror.Append(result, err)
	}

	if err := h.presence.Ready(); err != nil {
		result = multierror.Append(result, err)
	}

	return
}

// OnClientConnected occurs when a new client connection is accepted
func (h *Hub) OnClientConnected(sock *websocket.Conn) error {
	var id ClientID = 1234
	c := NewClient(id, sock, h)

	if err := h.presence.Add(c.id); err != nil {
		return err
	}

	h.register <- c
	c.Process()
	return nil
}

// registerClient registers client to hub
func (h *Hub) registerClient(c *Client) {
	h.clients[c.id] = c
}

// unregisterClient unregisters client from hub
func (h *Hub) unregisterClient(c *Client) {
	client, ok := h.clients[c.id]

	if ok && client.session == c.session {
		delete(h.clients, c.id)
	}
}

// broadcast distributes message with broker
func (h *Hub) broadcast(msg *Message) error {
	locations, err := h.presence.Locate(msg.GetRecipients())

	if err != nil {
		return err
	}

	for location, members := range locations {
		msg.SetRecipients(members)

		data, err := msg.GetBytes()

		if err != nil {
			continue
		}

		h.broker.Send(location, data)
	}

	return nil
}

// send sends message to recipients
func (h *Hub) send(msg *Message) error {
	data, err := msg.GetOutbound()

	if err != nil {
		return err
	}

	for _, id := range msg.GetRecipients() {
		client, ok := h.clients[id]

		if ok {
			select {
			case client.outbound <- data:
			default:
				close(client.outbound)
				delete(h.clients, client.id)
			}
		}
	}

	return nil
}
