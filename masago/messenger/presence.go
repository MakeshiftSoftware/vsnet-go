package messenger

import (
	"github.com/garyburd/redigo/redis"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
)

// PresenceRedisField type
type PresenceRedisField string

// Presence redis fields enum
const (
	// LocationField field
	LocationField PresenceRedisField = "l"
)

// Presence presence implementation
type Presence struct {
	name  string
	redis *predis.Client
}

// NewPresence creates new presence
func NewPresence(name string, redisAddr string) *Presence {
	return &Presence{
		name:  name,
		redis: predis.New(redisAddr),
	}
}

// Start starts presence
func (p *Presence) Start() error {
	return p.redis.WaitForConnection()
}

// Stop stops presence
func (p *Presence) Stop() error {
	return p.redis.Close()
}

// Ready checks if presence is ready
func (p *Presence) Ready() error {
	return p.redis.Ping()
}

// Add adds client to presence
func (p *Presence) Add(id ClientID) error {
	args := []interface{}{
		id,
		LocationField, p.name,
	}

	return p.redis.Hmset(args)
}

// Remove removes client from presence
func (p *Presence) Remove(id ClientID) error {
	return p.redis.Delete(id)
}

// RemoveMulti removes multiple clients from presence
func (p *Presence) RemoveMulti(ids []ClientID) error {
	con := p.redis.Pool.Get()
	defer con.Close()

	if err := con.Send("MULTI"); err != nil {
		return err
	}

	for id := range ids {
		if err := con.Send("DEL", id); err != nil {
			return err
		}
	}

	_, err := con.Do("EXEC")
	return err
}

// Locate locates clients by id
func (p *Presence) Locate(clients []ClientID) (map[string][]ClientID, error) {
	con := p.redis.Pool.Get()
	defer con.Close()

	locations := make(map[string][]ClientID)

	if err := con.Send("MULTI"); err != nil {
		return locations, err
	}

	for _, client := range clients {
		if err := con.Send("HGET", client, LocationField); err != nil {
			return locations, err
		}
	}

	result, err := redis.Strings(con.Do("EXEC"))

	if err != nil {
		return locations, err
	}

	for i, location := range result {
		locations[location] = append(locations[location], clients[i])
	}

	return locations, nil
}
