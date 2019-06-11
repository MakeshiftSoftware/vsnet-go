package node

import (
	"log"
	"sync"

	"github.com/garyburd/redigo/redis"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
)

const (
	peerPrefix   = "peer:"   // Prefix for peer message queue in redis
	masterPrefix = "master:" // Prefix for master message queue in redis
)

// Transport implementation
type Transport struct {
	wg      sync.WaitGroup  // Wait group
	redis   *predis.Client  // Redis client
	id      string          // Node ID
	masterc chan<- []byte   // Master message channel
	peerc   chan<- *Message // Peer message channel
	quitc   chan struct{}   // Quit channel
}

// newTransport creates new transport
func newTransport(id string, redis *predis.Client, masterc chan<- []byte, peerc chan<- *Message) *Transport {
	return &Transport{
		redis:   redis,
		id:      id,
		masterc: masterc,
		peerc:   peerc,
		quitc:   make(chan struct{}, 1),
	}
}

// start starts transport
func (t *Transport) start() error {
	log.Print("[info] starting transport...")

	// Start peer message consumer
	if err := t.listen(peerPrefix+t.id, t.receivePeer); err != nil {
		return err
	}

	// Start master message consumer
	if err := t.listen(masterPrefix+t.id, t.receiveMaster); err != nil {
		return err
	}

	log.Print("[info] transport started")

	return nil
}

// stop stops transport
func (t *Transport) stop() error {
	log.Print("[info] stopping transport...")

	// Initiate transport shutdown by closing quit channel
	close(t.quitc)

	// Wait for transport to stop
	t.wg.Wait()

	log.Print("[info] stopped transport")

	return nil
}

// Send sends data to node by id
func (t *Transport) send(id string, data []byte) error {
	// Push data into peer queue by id
	return t.redis.Rpush(peerPrefix+id, data)
}

// receivePeer handles message received from peer message queue
func (t *Transport) receivePeer(data []byte) error {
	msg, err := MessageFromBytes(data)
	t.peerc <- msg
	return err
}

// receiveMaster handles message received from master message queue
func (t *Transport) receiveMaster(data []byte) error {
	t.masterc <- data
	return nil
}

// listen starts message queue listener on key
func (t *Transport) listen(key string, receive func(data []byte) error) error {
	conn, err := t.redis.Pool.Dial()

	if err != nil {
		return err
	}

	// Listen for quit
	go func() {
		<-t.quitc
		conn.Close()
		return
	}()

	go func() {
		t.wg.Add(1)

		defer func() {
			conn.Close()
			t.wg.Done()
		}()

		for {
			data, err := redis.ByteSlices(conn.Do("BLPOP", key, 0))

			if err != nil {
				return
			}

			log.Printf("[info] received data from transport: %v", data[1])

			if err := receive(data[1]); err != nil {
				return
			}
		}
	}()

	return nil
}
