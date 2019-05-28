package messenger

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

// ClientID type
type ClientID uint64

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
	id       ClientID
	session  string
	hub      *Hub
	sock     *websocket.Conn
	outbound chan []byte
}

// NewClient creates a new client
func NewClient(id ClientID, sock *websocket.Conn, hub *Hub) *Client {
	log.Printf("[info] creating new client with id %d", id)

	return &Client{
		id:       id,
		session:  uuid.NewV4().String(),
		sock:     sock,
		hub:      hub,
		outbound: make(chan []byte, 256),
	}
}

// Process processes a client
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
				log.Printf("[error] socket close error: %v", err)
			}

			break
		}

		msg, err := MessageFromBytes(data)

		if err != nil {
			log.Printf("[error] error processing data: %v", err)
			break
		}

		msg.SetSender(c.id)
		c.hub.inbound <- msg
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

	for {
		select {
		case data, ok := <-c.outbound:
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

// sendMessage sends data through socket
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

// serveWs handles websocket connection requests
func serveWs(s *Service, w http.ResponseWriter, r *http.Request) error {
	log.Print("[info] received ws connection request")

	sock, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		return err
	}

	log.Print("[info] connection request upgraded")

	s.hub.OnClientConnected(sock)
	return nil
}
