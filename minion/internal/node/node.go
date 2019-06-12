package node

import (
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/makeshiftsoftware/vsnet/minion/internal/config"
	"github.com/makeshiftsoftware/vsnet/pkg/grace"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
	"github.com/makeshiftsoftware/vsnet/pkg/task"
	uuid "github.com/satori/go.uuid"
)

const (
	nodePrefix         = "minion:"       // Prefix for minion node in redis
	checkinPeriod      = 5 * time.Second // Keep node alive with this period
	nodeKeyExpires     = 10              // Time (in seconds) to expire node key
	nodeIPKey          = "ip"            // Key used to store Node IP
	nodePortKey        = "port"          // Key used to store Node port
	nodeConnectionsKey = "connections"   // Key used to store Node connections count
)

// ErrMinionNotFound is returned when the minion is not found in redis.
var ErrMinionNotFound = errors.New("could not find the requested minion")

// node implementation
type node struct {
	once     sync.Once
	wg       sync.WaitGroup
	cfg      *config.Config // Node config
	id       string         // Node ID
	redis    *predis.Client // Redis client
	http     *http.Server   // HTTP server
	hub      *hub           // Node hub
	quitc    chan os.Signal // Quit channel
	cleanupc chan struct{}  // Cleanup channel
}

// New creates a new node.
func New(cfg *config.Config) *node {
	n := &node{
		cfg:      cfg,
		id:       uuid.NewV4().String(),
		redis:    predis.New(cfg.RedisAddr),
		quitc:    make(chan os.Signal, 1),
		cleanupc: make(chan struct{}, 1),
	}

	n.hub = newHub(n.id, n.redis)

	n.initServer()

	return n
}

// Start starts the node.
func (n *node) Start() error {
	log.Print("[info] starting node...")

	// Setup graceful shutdown
	grace.HookSignals(n.quitc, n.Cleanup)

	log.Print("[info] connecting to redis...")

	// Wait for redis connection
	if err := n.redis.WaitForConnection(); err != nil {
		return err
	}

	log.Print("[info] connected to redis")

	// Start hub
	if err := n.hub.start(); err != nil {
		return err
	}

	// Join cluster
	if err := n.join(); err != nil {
		return err
	}

	// Start check-in task
	task.New(n.checkin, checkinPeriod, &n.wg, n.cleanupc)

	log.Printf("[info] node listening on port %s", n.cfg.Port)

	// Start http server
	return n.http.ListenAndServe()
}

// Cleanup disposes of node resources to shutdown server gracefully.
func (n *node) Cleanup() {
	n.once.Do(func() {
		log.Print("[info] cleaning up...")

		// Initiate cleanup by closing cleanup channel
		close(n.cleanupc)

		// Wait for tasks to stop
		n.wg.Wait()

		// Stop hub
		n.hub.stop()

		// Leave cluster
		if err := n.leave(); err != nil {
			log.Printf("[error] error leaving cluster: %v", err)
		}

		// Close redis
		if err := n.redis.Close(); err != nil {
			log.Printf("[error] error closing redis connection: %v", err)
		}

		log.Print("[info] finished cleanup")
	})
}

// join joins the minion node cluster by registering self to redis.
func (n *node) join() error {
	log.Print("[info] joining cluster...")

	conn := n.redis.Pool.Get()
	defer conn.Close()

	if err := conn.Send("MULTI"); err != nil {
		return err
	}

	// Initialize minion node in redis
	if err := conn.Send(
		"HMSET",
		nodePrefix+n.id,
		nodeIPKey, n.cfg.ExternalIP,
		nodePortKey, n.cfg.Port,
		nodeConnectionsKey, 0,
	); err != nil {
		return err
	}

	// Set minion node key expiration
	if err := conn.Send("EXPIRE", nodePrefix+n.id, nodeKeyExpires); err != nil {
		return err
	}

	if _, err := conn.Do("EXEC"); err != nil {
		return err
	}

	log.Print("[info] joined cluster")

	return nil
}

// leave leaves minion node cluster by deleting self from redis.
func (n *node) leave() error {
	log.Print("[info] leaving cluster...")
	// Delete minion node from redis
	return n.redis.Delete(nodePrefix + n.id)
}

// checkin keeps minion node in the cluster by extending node key expiration.
// If the node goes down, the node key will expire and the node will be treated
// as inactive.
func (n *node) checkin() bool {
	// Extend node key expiration in redis
	ok, err := n.redis.Expire(nodePrefix+n.id, nodeKeyExpires)

	if err != nil {
		log.Printf("[error] error refreshing node key: %v", err)
		return false
	}

	if !ok {
		log.Printf("[warn] node key does not exist")
		return false
	}

	return false
}

// initServer initializes the http server for the node.
func (n *node) initServer() {
	r := mux.NewRouter()

	r.HandleFunc("/healthz", n.wrapMiddleware(healthcheckHandler)).Methods("GET")
	r.HandleFunc("/ws", n.wrapMiddleware(serveWs)).Methods("GET")

	n.http = &http.Server{
		Handler: r,
		Addr:    ":" + n.cfg.Port,
	}
}
