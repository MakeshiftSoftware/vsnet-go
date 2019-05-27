package messenger

import (
	"log"

	"github.com/garyburd/redigo/redis"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
)

// MessageQueue implementation
type MessageQueue struct {
	key   string
	queue *predis.Client
	quit  chan struct{}
}

// NewMessageQueue creates a new message queue
func NewMessageQueue(key string, redisAddr string) *MessageQueue {
	return &MessageQueue{
		key:   key,
		queue: predis.NewClient(redisAddr),
		quit:  make(chan struct{}),
	}
}

// Start starts the message queue
func (mq *MessageQueue) Start() error {
	if err := mq.queue.WaitForConnection(); err != nil {
		return err
	}

	con := mq.queue.Pool.Get()

	go func() {
		for {
			select {
			case <-mq.quit:
				con.Close()
				return
			}
		}
	}()

	go func() {
		defer con.Close()

		for {
			data, err := redis.ByteSlices(con.Do("BLPOP", mq.key, 0))

			if err != nil {
				return
			}
			log.Printf("[info] received message from queue: %s", data[1])
		}
	}()
	return nil
}

// Stop stops message queue
func (mq *MessageQueue) Stop() {
	// delete queue key to clear pending messages
	mq.queue.Delete(mq.key)
	close(mq.quit)
	mq.queue.Close()
}

// Ready checks if service is ready
func (mq *MessageQueue) Ready() error {
	return mq.queue.Ping()
}
