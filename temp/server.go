package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	predis "github.com/makeshiftsoftware/vsnet/pkg/redis"
	"github.com/makeshiftsoftware/vsnet/pkg/utils"
	"github.com/pkg/errors"
)

// Server implementation
type Server struct {
	id           string         // Server id
	srv          *http.Server   // Http server
	hub          *Hub           // Server messaging hub
	gameState    *predis.Client // Redis client for game state
	presence     *predis.Client // Redis client for client presence
	messageQueue *predis.Client // Redis client for message queue
	nodeRegistry *predis.Client // Redis client for registering server
}

type handler func(*Server, http.ResponseWriter, *http.Request) error

func newServer() (*Server, error) {
	id := "server-" + utils.GenerateUUID4()

	log.Printf("[Info] Starting %v at %v:%v", id, cfg.ip, cfg.port)

	s := &Server{
		id:           id,
		hub:          newHub(),
		gameState:    predis.NewClient(cfg.gameStateService),
		presence:     predis.NewClient(cfg.presenceService),
		messageQueue: predis.NewClient(cfg.messageQueueService),
		nodeRegistry: predis.NewClient(cfg.nodeRegistryService),
	}

	r := s.newHandler()

	s.srv = &http.Server{
		Handler: r,
		Addr:    ":" + cfg.port,
	}

	return s, nil
}

func (s *Server) start() error {
	if err := s.gameState.WaitForConnection(); err != nil {
		return errors.Wrap(err, "Could not connect to redis game state")
	}

	if err := s.presence.WaitForConnection(); err != nil {
		return errors.Wrap(err, "Could not connect to redis presence")
	}

	if err := s.messageQueue.WaitForConnection(); err != nil {
		return errors.Wrap(err, "Could not connect to redis message queue")
	}

	if err := s.nodeRegistry.WaitForConnection(); err != nil {
		return errors.Wrap(err, "Could not connect to redis node registry")
	}

	if err := s.registerNode(); err != nil {
		return err
	}

	s.cleanupHook()

	return errors.Wrap(s.srv.ListenAndServe(), "Error starting server")
}

func (s *Server) cleanupHook() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGKILL)
	go func() {
		<-c
		s.gameState.Close()
		s.presence.Close()
		s.messageQueue.Close()
		s.nodeRegistry.Close()
		os.Exit(0)
	}()
}

func (s *Server) consumeMessages() {
	done := make(chan struct{})

	go func() {
		con := s.messageQueue.pool.Get()
		defer con.Close()

		for {
			select {
			case <-done:
				return
			default:
				message, err := redis.ByteSlices(con.Do("BLPOP", key, 0))

				if err == redis.ErrNil {
					// timeout
					continue
				}

				if err != nil {
					log.Printf("[Error] Could not read message from queue: %+v", err)
				}

				// process result of BRPOP
				log.Printf("[Info] Received message %s", message)
			}
		}
	}()

	time.Sleep(time.Second)
	close(done)
}

func (s *Server) newHandler() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/readiness", s.wrapMiddleware(readinessCheck)).Methods("GET")
	r.HandleFunc("/ws", s.wrapMiddleware(newConn)).Methods("GET")

	return r
}

func (s *Server) wrapMiddleware(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(s, w, r)

		if err != nil {
			log.Printf("[Error] %+v", err)

			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
