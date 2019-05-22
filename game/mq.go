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
		defer con.Close()

		for {
			select {
			case <-mq.quit:
				return
			default:
				message, err := redis.ByteSlices(con.Do("BLPOP", mq.key, 0))

				if err != nil {
					log.Printf("[Error] Could not read message from queue: %+v", err)
				}

				log.Printf("[Info] Received message from queue: %s", message[1])
			}
		}
	}()
	return
}

// Stop stops message queue
func (mq *MessageQueue) Stop() {
	close(mq.quit)
	mq.queue.Close()
}
