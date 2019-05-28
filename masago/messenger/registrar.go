package messenger

import (
	"log"

	"github.com/garyburd/redigo/redis"
	"github.com/hashicorp/go-multierror"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
	"github.com/pkg/errors"
)

const redisNodePrefix = "Node:"

var errNodeNotFound = errors.New("Node not found in registry")

// RegistrarRedisField type
type RegistrarRedisField string

// Registrar redis fields enum
const (
	// IDField field
	IDField = "id"
	// IPField field
	IPField = "ip"
	// PortField field
	PortField = "port"
	// ConnectionsField field
	ConnectionsField RegistrarRedisField = "connections"
)

// Registrar implementation
type Registrar struct {
	key      string
	id       string
	ip       string
	port     string
	registry *predis.Client
}

// Node implementation
type Node struct {
	id          string `redis:"id"`
	ip          string `redis:"ip"`
	port        string `redis:"port"`
	connections uint32 `redis:"connections"`
}

// NewRegistrar creates a new registrar
func NewRegistrar(id string, ip string, port string, redisAddr string) *Registrar {
	return &Registrar{
		key:      redisNodePrefix + id,
		id:       id,
		ip:       ip,
		port:     port,
		registry: predis.New(redisAddr),
	}
}

// Start starts the registrar
func (r *Registrar) Start() error {
	log.Print("[info] starting registrar...")

	if err := r.registry.WaitForConnection(); err != nil {
		return err
	}

	if err := r.registerNode(); err != nil {
		return err
	}

	log.Print("[info] regisrar started")
	return nil
}

// Stop stops the registrar
func (r *Registrar) Stop() (result error) {
	if err := r.unregisterNode(); err != nil {
		result = multierror.Append(result, err)
	}

	if err := r.registry.Close(); err != nil {
		result = multierror.Append(result, err)
	}

	return
}

// Ready checks if registrar is ready
func (r *Registrar) Ready() error {
	ok, err := r.registry.Exists(r.key)

	if err != nil {
		return err
	}

	if !ok {
		return errNodeNotFound
	}

	return nil
}

// GetNode gets node from registry
func (r *Registrar) GetNode() (Node, error) {
	var node Node
	values, err := r.registry.Hgetall(r.key)

	if err != nil {
		return node, err
	}

	if len(values) == 0 {
		return node, errNodeNotFound
	}

	err = redis.ScanStruct(values, &node)
	return node, err
}

// IncrementConnections increment connections by one
func (r *Registrar) IncrementConnections() error {
	_, err := r.registry.Hincrby(r.key, ConnectionsField, 1)
	return err
}

// GetConnections get connections count
func (r *Registrar) GetConnections() (result int, err error) {
	return r.registry.HgetInt(r.key, ConnectionsField)
}

// registerNode adds node to registry
func (r *Registrar) registerNode() error {
	args := []interface{}{
		r.key,
		IDField, r.id,
		IPField, r.ip,
		PortField, r.port,
		ConnectionsField, 0,
	}

	return r.registry.Hmset(args)
}

// unregisterNode removes node from registry
func (r *Registrar) unregisterNode() error {
	return r.registry.Delete(r.key)
}
