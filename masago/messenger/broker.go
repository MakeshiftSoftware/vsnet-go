package messenger

import (
	"log"

	"github.com/garyburd/redigo/redis"
	"github.com/hashicorp/go-multierror"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
)

// Broker implementation
type Broker struct {
	key       string
	redis     *predis.Client
	onReceive func(msg *Message) error
	quit      chan struct{}
}

// NewBroker creates new broker
func NewBroker(key string, redisAddr string, onReceive func(msg *Message) error) *Broker {
	return &Broker{
		key:   key,
		redis: predis.New(redisAddr),
		quit:  make(chan struct{}),
	}
}

// Start starts broker
func (b *Broker) Start() error {
	if err := b.redis.WaitForConnection(); err != nil {
		return err
	}

	con := b.redis.Pool.Get()

	go func() {
		for {
			select {
			case <-b.quit:
				con.Close()
				return
			}
		}
	}()

	go func() {
		defer con.Close()

		for {
			data, err := redis.ByteSlices(con.Do("BLPOP", b.key, 0))

			if err != nil {
				return
			}

			log.Printf("[info] broker received message: %s", data[1])

			msg, err := MessageFromBytes(data[1])

			if err != nil {
				return
			}

			b.onReceive(msg)
		}
	}()

	return nil
}

// Stop stops broker
func (b *Broker) Stop() (result error) {
	close(b.quit)

	// delete queue key to clear pending messages
	if err := b.redis.Delete(b.key); err != nil {
		result = multierror.Append(result, err)
	}

	if err := b.redis.Close(); err != nil {
		result = multierror.Append(result, err)
	}

	return
}

// Ready checks if broker is ready
func (b *Broker) Ready() error {
	return b.redis.Ping()
}

// Send sends message
func (b *Broker) Send(key PresenceLocation, data []byte) error {
	return b.redis.Rpush(key, data)
}
