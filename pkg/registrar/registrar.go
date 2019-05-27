package registrar

import (
	"github.com/garyburd/redigo/redis"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
	"github.com/pkg/errors"
)

const redisNodePrefix = "Node:"

var errNodeNotFound = errors.New("Node not found in registry")

// Registrar implementation
type Registrar struct {
	key      string
	node     *Node
	registry *predis.Client
}

// Node implementation
type Node struct {
	id   string `redis:"id"`
	ip   string `redis:"ip"`
	port string `redis:"port"`
}

// New creates a new registrar
func New(id string, ip string, port string, redisAddr string) *Registrar {
	return &Registrar{
		key: redisNodePrefix + id,
		node: &Node{
			id:   id,
			ip:   ip,
			port: port,
		},
		registry: predis.NewClient(redisAddr),
	}
}

// Start starts the registrar
func (r *Registrar) Start() error {
	if err := r.registry.WaitForConnection(); err != nil {
		return err
	}

	if err := r.registerNode(); err != nil {
		return err
	}
	return nil
}

// Stop stops the registrar
func (r *Registrar) Stop() {
	r.unregisterNode()
	r.registry.Close()
}

// Ready checks if registrar is ready
func (r *Registrar) Ready() error {
	if err := r.registry.Ping(); err != nil {
		return err
	}

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

// registerNode adds node to registry
func (r *Registrar) registerNode() error {
	args := []interface{}{
		r.key,
		"id", r.node.id,
		"ip", r.node.ip,
		"port", r.node.port,
	}
	return r.registry.Hmset(args)
}

// unregisterNode removes node from registry
func (r *Registrar) unregisterNode() error {
	return r.registry.Delete(r.key)
}
