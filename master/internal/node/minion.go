package node

import (
	"log"
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
)

const (
	minionPrefix  = "minion:"
	messagePrefix = "master:"
)

// ErrMinionNotFound is returned when the minion is not found in redis.
var ErrMinionNotFound = errors.New("could not find the requested minion")

// minion implementation
type minion struct {
	ID          string `redis:"id" json:"id"`                   // Minion ID
	IP          string `redis:"ip" json:"ip"`                   // Minion external IP
	Port        string `redis:"port" json:"port"`               // Minion port
	Connections uint64 `redis:"connections" json:"connections"` // Minion connections count
}

// getMinionKeys retrieves all keys for active minions in redis.
func (n *node) getMinionKeys() ([]string, error) {
	return n.redis.GetKeys(minionPrefix + "*")
}

// getMinions retrieves all active minions from redis.
func (n *node) getMinions() (result []minion, err error) {
	conn := n.redis.Pool.Get()
	defer conn.Close()

	var keys []string

	keys, err = n.getMinionKeys()

	if err != nil {
		return result, err
	}

	if err = conn.Send("MULTI"); err != nil {
		return result, err
	}

	for _, key := range keys {
		if err = conn.Send("HGETALL", key); err != nil {
			return result, err
		}
	}

	values, err := redis.Values(conn.Do("EXEC"))

	if err != nil {
		return result, err
	}

	for _, val := range values {
		var m minion

		if len(val.([]interface{})) == 0 {
			return result, ErrMinionNotFound
		}

		if err := redis.ScanStruct(val.([]interface{}), &m); err != nil {
			return result, err
		}

		result = append(result, m)
	}

	return result, nil
}

// getMinion retrieves a minion by its id.
func (n *node) getMinion(id string) (result minion, err error) {
	values, err := redis.Values(n.redis.Hgetall(minionPrefix + id))

	if err != nil {
		return result, err
	}

	if len(values) == 0 {
		return result, ErrMinionNotFound
	}

	err = redis.ScanStruct(values, &result)

	return result, err
}

// sendMessage sends a message to a specific minion by its id.
func (n *node) sendMessage(id string, data []byte) error {
	ok, err := n.redis.Exists(minionPrefix + id)

	if err != nil {
		return err
	}

	if !ok {
		log.Printf("[warn] minion with id %s does not exist", id)
		return ErrMinionNotFound
	}

	return n.redis.Rpush(messagePrefix+id, data)
}

// broadcastMessage broadcasts a message to all active minions.
func (n *node) broadcastMessage(data []byte) (err error) {
	conn := n.redis.Pool.Get()
	defer conn.Close()

	var keys []string

	keys, err = n.getMinionKeys()

	if err = conn.Send("MULTI"); err != nil {
		return err
	}

	for _, key := range keys {
		id := strings.Split(key, ":")[1]

		if err = conn.Send("RPUSH", messagePrefix+id, data); err != nil {
			return err
		}
	}

	if _, err = conn.Do("EXEC"); err != nil {
		return err
	}

	return nil
}
