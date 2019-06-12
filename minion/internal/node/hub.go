package node

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
)

// hub implementation
type hub struct {
	sync.RWMutex
	id          string             // Node ID
	redis       *predis.Client     // Redis client
	presence    *presence          // Hub presence
	transport   *transport         // Hub transport
	clients     map[string]*client // Connected clients
	inboundc    chan *Message      // Client inbound message channel
	masterc     chan []byte        // Master message channel
	peerc       chan *Message      // Peer message channel
	registerc   chan *client       // Register channel
	unregisterc chan *client       // Unregister channel
}

// newHub creates a new hub.
func newHub(id string, redis *predis.Client) *hub {
	h := &hub{
		id:          id,
		redis:       redis,
		clients:     make(map[string]*client),
		inboundc:    make(chan *Message),
		peerc:       make(chan *Message),
		masterc:     make(chan []byte),
		registerc:   make(chan *client),
		unregisterc: make(chan *client),
	}

	h.presence = newPresence(h.id, h.redis)
	h.transport = newTransport(h.id, h.redis, h.masterc, h.peerc)

	return h
}

// start starts the hub.
func (h *hub) start() error {
	// Start transport
	if err := h.transport.start(); err != nil {
		return err
	}

	// Start hub channel listeners
	go func() {
		for {
			select {
			case client := <-h.registerc:
				// Handle register client request
				h.registerClient(client)
			case client := <-h.unregisterc:
				// Handle unregister client request
				h.unregisterClient(client)
			case msg := <-h.inboundc:
				// Handle message received from client
				h.onClientMessage(msg)
			case msg := <-h.peerc:
				// Handle message received from peer
				h.onPeerMessage(msg)
			case data := <-h.masterc:
				// Handle message received from master
				h.onMasterMessage(data)
			}
		}
	}()

	return nil
}

// stop stops the hub.
func (h *hub) stop() {
	h.Lock()
	defer h.Unlock()

	// Stop transport
	if err := h.transport.stop(); err != nil {
		log.Printf("[error] error stopping transport: %v", err)
	}

	// Create list of clients ids to be terminated
	ids := make([]string, len(h.clients))
	i := 0

	// Close client connections
	for id, client := range h.clients {
		close(client.outboundc)
		client.sock.Close()
		ids[i] = id
		i++
	}

	// Remove clients from presence
	if err := h.presence.removeMulti(ids); err != nil {
		log.Printf("[error] error removing clients from presence: %v", err)
	}
}

// registerClient registers a client to the hub.
func (h *hub) registerClient(c *client) {
	h.Lock()
	defer h.Unlock()

	// Add client id to connected clients map
	h.clients[c.id] = c
}

// unregisterClient unregisters a client from the hub.
func (h *hub) unregisterClient(c *client) {
	h.Lock()
	defer h.Unlock()

	// Check if client exists in hub
	client, ok := h.clients[c.id]

	// Check if client's session id is the same as the client
	// being unregistered.
	if ok && client.sess == c.sess {
		delete(h.clients, c.id)
	}
}

// onClientConnected handles a new client socket connection.
func (h *hub) onClientConnected(sock *websocket.Conn) error {
	// Create new client
	c := newClient("abc", h, sock)

	// Add client to presence
	if err := h.presence.add(c.id); err != nil {
		return err
	}

	// Register client using channel
	h.registerc <- c

	// Start client processes
	c.process()
	return nil
}

// onClientMessage handles messages received from client. Routes messages received
// to their intended recipients on remote minion nodes.
func (h *hub) onClientMessage(msg *Message) error {
	locations, err := h.presence.locate(msg.GetRecipients())

	if err != nil {
		return err
	}

	for location, members := range locations {
		msg.SetRecipients(members)

		data, err := msg.GetBytes()

		if err != nil {
			continue
		}

		h.transport.send(location, data)
	}

	return nil
}

// onPeerMessage handles messages received from peer nodes. Routes message to intended
// recipients on the local minion node.
func (h *hub) onPeerMessage(msg *Message) error {
	// Get outbound message for delivery
	data, err := msg.GetOutbound()

	if err != nil {
		return err
	}

	for _, id := range msg.GetRecipients() {
		// Find client on this node
		client, ok := h.clients[id]

		if ok {
			// Attempt message delivery
			select {
			case client.outboundc <- data:
			default:
				close(client.outboundc)
				delete(h.clients, client.id)
			}
		}
	}

	return nil
}

// onMasterMessage handles messages received from master node.
func (h *hub) onMasterMessage(data []byte) error {
	return nil
}
