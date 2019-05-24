package main

// Message implementation
type Message struct {
	sender *Client
	data   []byte
}

// NewMessage creates a new message
func NewMessage(sender *Client, data []byte) *Message {
	return &Message{
		sender: sender,
		data:   data,
	}
}
