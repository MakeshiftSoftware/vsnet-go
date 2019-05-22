package main

import (
	"bytes"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/makeshiftsoftware/vsnet/pkg/auth"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client implementation
type Client struct {
	accessKey *auth.AccessKey // Client access key
	hub       *Hub            // Server hub
	ws        *websocket.Conn // Websocket connection
	send      chan []byte     // Buffered channel of outbound messages
}

func newClient(key *auth.AccessKey, hub *Hub, ws *websocket.Conn) *Client {
	return &Client{
		accessKey: key,
		hub:       hub,
		ws:        ws,
		send:      make(chan []byte, 256),
	}
}

// readPump pumps messages from the websocket connection to the hub
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()

	c.ws.SetReadLimit(cfg.maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(cfg.pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(cfg.pongWait))

		return nil
	})

	for {
		_, message, err := c.ws.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Error] Socket close error: %v", err)
			}

			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(cfg.pingPeriod)

	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(cfg.writeWait))

			if !ok {
				// The hub closed the channel
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.ws.NextWriter(websocket.TextMessage)

			if err != nil {
				return
			}

			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)

			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(cfg.writeWait))

			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) getID() string {
	return c.accessKey.ID
}
