package node

import (
	"github.com/garyburd/redigo/redis"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
)

const (
	clientPrefix = "client:"
)

// presence implementation
type presence struct {
	id    string         // Node ID
	redis *predis.Client // Redis client
}

// newPresence creates a new presence.
func newPresence(id string, redis *predis.Client) *presence {
	return &presence{
		id:    id,
		redis: redis,
	}
}

// add adds a client to presence by its client id.
func (p *presence) add(id string) error {
	_, err := p.redis.Set(clientPrefix+id, p.id)
	return err
}

// remove removes a client from presence by its client id.
func (p *presence) remove(id string) error {
	return p.redis.Delete(clientPrefix + id)
}

// removeMulti removes multiple clients from presence given an array of client ids.
func (p *presence) removeMulti(ids []string) error {
	con := p.redis.Pool.Get()
	defer con.Close()

	if err := con.Send("MULTI"); err != nil {
		return err
	}

	for _, id := range ids {
		if err := con.Send("DEL", clientPrefix+id); err != nil {
			return err
		}
	}

	_, err := con.Do("EXEC")
	return err
}

// locate finds node locations of clients given an array of client ids.
// The result will be a map where each key is a minion id and each value
// is an array of client ids from the original client id array that exist
// on that minion node.
func (p *presence) locate(ids []string) (map[string][]string, error) {
	conn := p.redis.Pool.Get()
	defer conn.Close()

	locations := make(map[string][]string)

	if err := conn.Send("MULTI"); err != nil {
		return locations, err
	}

	for _, id := range ids {
		if err := conn.Send("GET", clientPrefix+id); err != nil {
			return locations, err
		}
	}

	result, err := redis.Strings(conn.Do("EXEC"))

	if err != nil {
		return locations, err
	}

	for i, location := range result {
		locations[location] = append(locations[location], ids[i])
	}

	return locations, nil
}
