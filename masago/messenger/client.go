package messenger

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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

var newline = []byte{'\n'}

// Default upgrader to upgrade connections
var upgrader = &websocket.Upgrader{
	ReadBufferSize:  readBufferSize,
	WriteBufferSize: writeBufferSize,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client implementation
type Client struct {
	id   string
	hub  *Hub
	sock *websocket.Conn
	send chan *Message
}

// NewClient creates a new client
func NewClient(id string, sock *websocket.Conn, hub *Hub) *Client {
	return &Client{
		id:   id,
		sock: sock,
		hub:  hub,
		send: make(chan *Message, 256),
	}
}

// Process process clients
func (c *Client) Process() {
	c.sock.SetReadLimit(maxMessageSize)
	c.sock.SetReadDeadline(time.Now().Add(pongWait))
	c.sock.SetPongHandler(c.setReadDeadline)
	go c.Read()
	go c.Write()
}

// Read reads data from client socket
func (c *Client) Read() {
	defer func() {
		c.hub.unregister <- c
		c.sock.Close()
	}()

	for {
		_, data, err := c.sock.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Error] %v", err)
			}
			break
		}
		c.hub.inbound <- NewMessage(c, data)
	}
}

// Write writes data to client socket
func (c *Client) Write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.unregister <- c
		c.sock.Close()
	}()

	go func() {
		for {
			select {
			case msg, ok := <-c.send:
				c.setWriteDeadline()
				if !ok {
					// The hub closed the channel
					return
				}

				if err := c.sendMessage(msg); err != nil {
					return
				}
			case <-ticker.C:
				c.setWriteDeadline()
				if err := c.sock.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()
}

// sendMessage sends message through socket
func (c *Client) sendMessage(msg *Message) (err error) {
	var w io.WriteCloser
	w, err = c.sock.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return
	}
	w.Write(msg.data)

	// Combine queued messages with current message
	n := len(c.send)
	for i := 0; i < n; i++ {
		w.Write(newline)
		addMsg := <-c.send
		w.Write(addMsg.data)
	}

	err = w.Close()
	return
}

// setReadDeadline sets read deadline for client
func (c *Client) setReadDeadline(string) error {
	return c.sock.SetReadDeadline(time.Now().Add(pongWait))
}

// setWriteDeadline sets write deadline for client
func (c *Client) setWriteDeadline() {
	c.sock.SetWriteDeadline(time.Now().Add(writeWait))
}

// serveWs handles websocket connection requests
func serveWs(s *Service, w http.ResponseWriter, r *http.Request) (err error) {
	var sock *websocket.Conn
	if sock, err = upgrader.Upgrade(w, r, nil); err == nil {
		s.hub.OnClientConnected(sock)
	}
	return
}
