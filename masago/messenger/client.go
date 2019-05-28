package messenger

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
	hub      *Hub
	sock     *websocket.Conn
	outbound chan []byte
}

// NewClient creates a new client
func NewClient(id ClientID, sock *websocket.Conn, hub *Hub) *Client {
	return &Client{
		id:       id,
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

	go func() {
		defer func() {
			c.hub.unregister <- c
			c.sock.Close()
		}()

		var wg sync.WaitGroup
		go c.Read(&wg)
		go c.Write(&wg)
		wg.Wait()
	}()
}

// Read reads data from client socket
func (c *Client) Read(wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	for {
		_, data, err := c.sock.ReadMessage()

		if err != nil {
			break
		}

		msg, err := MessageFromBytes(data)

		if err != nil {
			break
		}

		msg.SetSender(c.id)
		c.hub.inbound <- msg
	}
}

// Write writes data to client socket
func (c *Client) Write(wg *sync.WaitGroup) {
	wg.Add(1)
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		wg.Done()
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

	// @TODO: figure out how to separate multiple binary messages in single send
	// Combine queued messages with current message
	// n := len(c.send)
	// for i := 0; i < n; i++ {
	// 	w.Write(newline)
	// 	addMsg := <-c.send
	// 	w.Write(addMsg.data)
	// }

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
	sock, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		return err
	}

	s.hub.OnClientConnected(sock)
	return nil
}
