package node

import (
	"time"

	"github.com/vmihailenco/msgpack"
)

// MessageType type
type MessageType uint8

// MessageType enum
const (
	// Chat message type
	Chat MessageType = iota
)

// TimestampRequired denotes message types that require a timestamp
var TimestampRequired = map[MessageType]struct{}{
	(Chat): struct{}{},
}

// IMessage interface
type IMessage interface {
	GetBytes() []byte
	GetOutbound() []byte
	GetType() MessageType
	SetType(t MessageType)
	GetData() []byte
	SetData(data []byte)
	GetSender() string
	SetSender(id string)
	GetRecipients() []string
	SetRecipients(ids []string)
	GetTimestamp() time.Time
	SetTimestamp()
}

// Message implementation
type Message struct {
	Type      MessageType `msgpack:"t,omitempty"`  // Message type
	Data      []byte      `msgpack:"d,omitempty"`  // Message data
	Sender    string      `msgpack:"s,omitempty"`  // Message sender
	Recipient []string    `msgpack:"r,omitempty"`  // Message recipients
	Timestamp time.Time   `msgpack:"ts,omitempty"` // Message timestamp
}

// MessageFromBytes creates a new message from raw bytes
func MessageFromBytes(data []byte) (*Message, error) {
	var msg Message
	err := msgpack.Unmarshal(data, &msg)
	return &msg, err
}

// GetBytes gets message bytes
func (msg *Message) GetBytes() ([]byte, error) {
	return msgpack.Marshal(msg)
}

// GetOutbound gets message bytes for outbound delivery
func (msg *Message) GetOutbound() ([]byte, error) {
	out := &Message{
		Type:   msg.GetType(),
		Data:   msg.GetData(),
		Sender: msg.GetSender(),
	}

	if _, ok := TimestampRequired[msg.GetType()]; ok {
		out.SetTimestamp()
	}

	return msgpack.Marshal(out)
}

// GetType gets message type
func (msg *Message) GetType() MessageType {
	return msg.Type
}

// SetType sets message type
func (msg *Message) SetType(t MessageType) {
	msg.Type = t
}

// GetData gets message data
func (msg *Message) GetData() []byte {
	return msg.Data
}

// SetData sets message data
func (msg *Message) SetData(d []byte) {
	msg.Data = d
}

// GetSender gets message sender
func (msg *Message) GetSender() string {
	return msg.Sender
}

// SetSender sets message sender
func (msg *Message) SetSender(id string) {
	msg.Sender = id
}

// GetRecipients gets message recipient
func (msg *Message) GetRecipients() []string {
	return msg.Recipient
}

// SetRecipients sets message recipient
func (msg *Message) SetRecipients(ids []string) {
	msg.Recipient = ids
}

// GetTimestamp gets message timestamp
func (msg *Message) GetTimestamp() time.Time {
	return msg.Timestamp
}

// SetTimestamp sets message timestamp
func (msg *Message) SetTimestamp() {
	msg.Timestamp = time.Now()
}
