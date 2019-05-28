package messenger

import (
	"time"

	"github.com/vmihailenco/msgpack"
)

// MessageBytes type
type MessageBytes []byte

// MessageType type
type MessageType uint8

// MessageData type
type MessageData []byte

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
	GetBytes() MessageBytes
	GetOutbound() MessageBytes
	GetType() MessageType
	SetType(t MessageType)
	GetData() MessageData
	SetData(data MessageData)
	GetSender() ClientID
	SetSender(id ClientID)
	GetRecipients() []ClientID
	SetRecipients(ids []ClientID)
	GetTimestamp() time.Time
	SetTimestamp()
}

// Message implementation
type Message struct {
	t  MessageType `msgpack:",omitempty"` // Message type
	d  MessageData `msgpack:",omitempty"` // Message data
	s  ClientID    `msgpack:",omitempty"` // Message sender
	r  []ClientID  `msgpack:",omitempty"` // Message recipients
	ts time.Time   `msgpack:",omitempty"` // Message timestamp
}

// MessageFromBytes creates a new message from raw bytes
func MessageFromBytes(data MessageBytes) (msg *Message, err error) {
	err = msgpack.Unmarshal(data, &msg)
	return
}

// GetBytes gets message bytes
func (msg *Message) GetBytes() (b MessageBytes, err error) {
	b, err = msgpack.Marshal(msg)
	return
}

// GetOutbound gets message bytes for outbound delivery
func (msg *Message) GetOutbound() (b MessageBytes, err error) {
	out := &Message{
		t: msg.GetType(),
		d: msg.GetData(),
		s: msg.GetSender(),
	}

	if _, ok := TimestampRequired[msg.GetType()]; ok {
		out.SetTimestamp()
	}

	b, err = msgpack.Marshal(out)
	return
}

// GetType gets message type
func (msg *Message) GetType() MessageType {
	return msg.t
}

// SetType sets message type
func (msg *Message) SetType(t MessageType) {
	msg.t = t
}

// GetData gets message data
func (msg *Message) GetData() MessageData {
	return msg.d
}

// SetData sets message data
func (msg *Message) SetData(d MessageData) {
	msg.d = d
}

// GetSender gets message sender
func (msg *Message) GetSender() ClientID {
	return msg.s
}

// SetSender sets message sender
func (msg *Message) SetSender(s ClientID) {
	msg.s = s
}

// GetRecipients gets message recipient
func (msg *Message) GetRecipients() []ClientID {
	return msg.r
}

// SetRecipients sets message recipient
func (msg *Message) SetRecipients(r []ClientID) {
	msg.r = r
}

// GetTimestamp gets message timestamp
func (msg *Message) GetTimestamp() time.Time {
	return msg.ts
}

// SetTimestamp sets message timestamp
func (msg *Message) SetTimestamp() {
	msg.ts = time.Now()
}
