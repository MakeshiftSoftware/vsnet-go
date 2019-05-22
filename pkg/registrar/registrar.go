package registrar

import (
	"context"
	"fmt"
	"sync"

	"github.com/garyburd/redigo/redis"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
	"github.com/pkg/errors"
)

const redisNodePrefix = "Node:"

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
func (r *Registrar) Start(ctx context.Context, wg *sync.WaitGroup) (err error) {
	err = r.registry.WaitForConnection()

	if err != nil {
		return
	}

	err = r.registerNode()
	return
}

// Stop stops registrar
func (r *Registrar) Stop() {
	r.unregisterNode()
	r.registry.Close()
}

// GetNode gets node from registry
func (r *Registrar) GetNode() (node Node, err error) {
	var values []interface{}
	values, err = r.registry.Hgetall(r.key)

	if err != nil {
		return
	}

	if len(values) == 0 {
		err = errors.New(fmt.Sprintf("Node %s not found in registry", r.key))
	} else {
		err = redis.ScanStruct(values, &node)
	}
	return
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
	return r.registry.Delete(r.node.id)
}
