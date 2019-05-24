package main

import (
	"context"
	"log"
	"sync"

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
func (mq *MessageQueue) Start(ctx context.Context, wg *sync.WaitGroup) (err error) {
	err = mq.queue.WaitForConnection()

	if err != nil {
		return
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
			message, err := redis.ByteSlices(con.Do("BLPOP", mq.key, 0))

			if err != nil {
				return
			}

			log.Printf("[Info] Received message from queue: %s", message[1])
		}
	}()
	return
}

// Stop stops message queue
func (mq *MessageQueue) Stop() {
	close(mq.quit)
	mq.flush()
	mq.queue.Close()
}

// flush deletes queue key to clear any pending messages
func (mq *MessageQueue) flush() {
	mq.queue.Delete(mq.key)
}
