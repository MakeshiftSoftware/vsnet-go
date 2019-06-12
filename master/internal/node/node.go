package node

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/makeshiftsoftware/vsnet/master/internal/config"
	"github.com/makeshiftsoftware/vsnet/pkg/grace"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
	"github.com/makeshiftsoftware/vsnet/pkg/task"
)

const (
	upgradePeriod    = 5 * time.Second // Attempt to upgrade node with this period
	maintainPeriod   = 5 * time.Second // Maintain control of master lock with this period
	masterKey        = "master"        // Key to use as master lock for node upgrading
	masterKeyExpires = 10              // Time (in seconds) to expire master lock key (should be longer than refreshPeriod)
)

// Node implementation
type Node struct {
	sync.RWMutex
	once     sync.Once
	wg       sync.WaitGroup
	cfg      *config.Config // Node config
	master   bool           // Node is master
	http     *http.Server   // HTTP server
	redis    *predis.Client // Redis client
	quitc    chan os.Signal // Quit channel
	cleanupc chan struct{}  // Cleanup channel
}

// New creates new node
func New(cfg *config.Config) *Node {
	n := &Node{
		cfg:      cfg,
		master:   false,
		redis:    predis.New(cfg.RedisAddr),
		quitc:    make(chan os.Signal, 1),
		cleanupc: make(chan struct{}, 1),
	}

	n.initServer()

	return n
}

// Start starts node
func (n *Node) Start() error {
	log.Print("[info] starting node...")

	grace.HookSignals(n.quitc, n.Cleanup)

	log.Print("[info] connecting to redis...")

	if err := n.redis.WaitForConnection(); err != nil {
		return err
	}

	log.Print("[info] connected to redis")

	task.New(n.upgrade, upgradePeriod, &n.wg, n.cleanupc)
	task.New(n.maintain, maintainPeriod, &n.wg, n.cleanupc)

	log.Printf("[info] node listening on port %s", n.cfg.Port)

	return n.http.ListenAndServe()
}

// Cleanup disposes of node resources
func (n *Node) Cleanup() {
	n.once.Do(func() {
		log.Print("[info] cleaning up...")

		close(n.cleanupc)
		n.wg.Wait()

		if err := n.redis.Close(); err != nil {
			log.Printf("[error] error closing redis connection: %v", err)
		}

		log.Print("[info] finished cleanup")
	})
}

// upgrade attempts to upgrade leader node to master node
func (n *Node) upgrade() bool {
	n.Lock()
	defer n.Unlock()

	if n.master {
		return false
	}

	log.Print("[info] attempting to upgrade node to master...")

	ok, err := n.redis.Set(
		masterKey, 1,
		"EX", masterKeyExpires,
		"NX",
	)

	if err != nil {
		log.Printf("[error] error acquiring master lock: %v", err)
		return false
	}

	if ok {
		log.Print("[info] acquired master lock, upgrading node to master...")
		n.master = true
		return false
	}

	return false
}

// maintain maintains control of master lock
func (n *Node) maintain() bool {
	n.Lock()
	defer n.Unlock()

	if !n.master {
		return false
	}

	ok, err := n.redis.Expire(masterKey, masterKeyExpires)

	if !ok || err != nil {
		n.master = false
	}

	if err != nil {
		log.Printf("[error] error extending master key expiration: %v", err)
		return false
	}

	if !ok {
		log.Printf("[warn] master key does not exist")
		return false
	}

	return false
}

// initServer initialize http server
func (n *Node) initServer() {
	r := mux.NewRouter()

	r.HandleFunc("/healthz", n.wrapMiddleware(healthcheckHandler)).Methods("GET")
	r.HandleFunc("/minions", n.wrapMiddleware(getMinionsHandler)).Methods("GET")
	r.HandleFunc("/minions/{id}", n.wrapMiddleware(getMinionHandler)).Methods("GET")
	r.HandleFunc("/minions/{id}/send", n.wrapMiddleware(sendMessageHandler)).Methods("POST")
	r.HandleFunc("/broadcast", n.wrapMiddleware(healthcheckHandler)).Methods("POST")

	n.http = &http.Server{
		Handler: r,
		Addr:    ":" + n.cfg.Port,
	}
}
