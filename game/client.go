package main

import (
	"log"
	"net/http"
	"sync"
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
	id   string          // Client id
	sock *websocket.Conn // The websocket connection
	send chan *Message   // Outbound messages channel
	hub  *Hub            // Hub
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
	defer func() {
		c.hub.unregister <- c
	}()

	var wg sync.WaitGroup
	c.Read(&wg)
	c.Write(&wg)
	wg.Wait()
}

// @TODO break this out into multiple funcs and make it more robust
// Read reads data from client socket
func (c *Client) Read(wg *sync.WaitGroup) {
	defer wg.Done()
	wg.Add(1)

	c.sock.SetReadLimit(maxMessageSize)
	c.sock.SetReadDeadline(time.Now().Add(pongWait))
	c.sock.SetPongHandler(c.setReadDeadline)

	go func() {
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
	}()
}

// @TODO break this out into multiple funcs
// Write writes data to client socket
func (c *Client) Write(wg *sync.WaitGroup) {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		wg.Done()
	}()
	wg.Add(1)

	go func() {
		for {
			select {
			case msg, ok := <-c.send:
				c.setWriteDeadline()
				if !ok {
					// The hub closed the channel
					c.sock.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}

				w, err := c.sock.NextWriter(websocket.BinaryMessage)
				if err != nil {
					return
				}
				w.Write(msg.data)

				// Combine queued messages with current message
				n := len(c.send)
				for i := 0; i < n; i++ {
					// @TODO Determine best way to separate multiple binary messages
					w.Write(newline)
					addMsg := <-c.send
					w.Write(addMsg.data)
				}

				if err := w.Close(); err != nil {
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

// setReadDeadline sets read deadline for client
func (c *Client) setReadDeadline(string) error {
	return c.sock.SetReadDeadline(time.Now().Add(pongWait))
}

// setWriteDeadline sets write deadline for client
func (c *Client) setWriteDeadline() {
	c.sock.SetWriteDeadline(time.Now().Add(writeWait))
}

// serveWs handles websocket connection requests and upgrades them
func serveWs(s *Server, w http.ResponseWriter, r *http.Request) (err error) {
	var sock *websocket.Conn
	sock, err = upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
	} else {
		s.hub.OnClientConnected(sock)
	}
	return
}
