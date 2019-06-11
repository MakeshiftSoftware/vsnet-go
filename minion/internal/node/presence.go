package node

import (
	"github.com/garyburd/redigo/redis"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
)

const (
	clientPrefix = "client:"
)

// Presence implementation
type Presence struct {
	id    string         // Node ID
	redis *predis.Client // Redis client
}

// newPresence creates new presence
func newPresence(id string, redis *predis.Client) *Presence {
	return &Presence{
		id:    id,
		redis: redis,
	}
}

// add adds client to presence
func (p *Presence) add(id string) error {
	_, err := p.redis.Set(clientPrefix+id, p.id)
	return err
}

// remove removes client from presence
func (p *Presence) remove(id string) error {
	return p.redis.Delete(clientPrefix + id)
}

// removeMulti removes multiple clients from presence
func (p *Presence) removeMulti(ids []string) error {
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

// locate locates clients by id
func (p *Presence) locate(ids []string) (map[string][]string, error) {
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
