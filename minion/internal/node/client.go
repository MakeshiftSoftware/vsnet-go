package node

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

const (
	readBufferSize   = 1024                // Size (bytes) of the read buffer
	writeBufferSize  = 1024                // Size (bytes) of the write buffer
	maxMessageSize   = 512                 // Maximum message size (bytes) allowed from client
	writeWait        = 10 * time.Second    // Time allowed to write a message to the client
	pongWait         = 60 * time.Second    // Time allowed to read the next pong message from the client
	pingPeriod       = (pongWait * 9) / 10 // Send pings to client with this period (must be less than pongWait)
	closeGracePeriod = 10 * time.Second    // Time to wait before force close a connection
)

// Upgrader default request upgrader
var upgrader = &websocket.Upgrader{
	ReadBufferSize:  readBufferSize,
	WriteBufferSize: writeBufferSize,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client implementation
type Client struct {
	sess      string          // Unique session ID
	id        string          // Unique client ID
	hub       *Hub            // Node hub
	sock      *websocket.Conn // Underlying socket connection
	outboundc chan []byte     // Client outbound message channel
}

// newClient creates a new client
func newClient(id string, hub *Hub, sock *websocket.Conn) *Client {
	return &Client{
		sess:      uuid.NewV4().String(),
		id:        id,
		hub:       hub,
		sock:      sock,
		outboundc: make(chan []byte, 256),
	}
}

// process processes a client
func (c *Client) process() {
	c.sock.SetReadLimit(maxMessageSize)
	c.sock.SetReadDeadline(time.Now().Add(pongWait))
	c.sock.SetPongHandler(c.setReadDeadline)
	go c.read()
	go c.write()
}

// read reads data from client socket
func (c *Client) read() {
	defer func() {
		c.hub.unregisterc <- c
		c.sock.Close()
	}()

	for {
		_, data, err := c.sock.ReadMessage()

		if err != nil {
			return
		}

		msg, err := MessageFromBytes(data)

		if err != nil {
			log.Printf("[error] error processing data: %v", err)
			return
		}

		msg.SetSender(c.id)
		c.hub.inboundc <- msg
	}
}

// write writes data to client socket
func (c *Client) write() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.hub.unregisterc <- c
		c.sock.Close()
	}()

	for {
		select {
		case data, ok := <-c.outboundc:
			c.setWriteDeadline()

			if !ok {
				// The hub closed the channel
				return
			}

			if err := c.send(data); err != nil {
				return
			}
		case <-ticker.C:
			c.setWriteDeadline()

			if err := c.sock.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// send sends data through socket
func (c *Client) send(data []byte) error {
	w, err := c.sock.NextWriter(websocket.BinaryMessage)

	if err != nil {
		return err
	}

	w.Write(data)

	return w.Close()
}

// setReadDeadline sets read deadline for client
func (c *Client) setReadDeadline(string) error {
	return c.sock.SetReadDeadline(time.Now().Add(pongWait))
}

// setWriteDeadline sets write deadline for client
func (c *Client) setWriteDeadline() error {
	return c.sock.SetWriteDeadline(time.Now().Add(writeWait))
}
