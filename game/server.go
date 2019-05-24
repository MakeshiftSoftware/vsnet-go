package main

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/makeshiftsoftware/vsnet/pkg/cleanup"
	"github.com/makeshiftsoftware/vsnet/pkg/registrar"
	uuid "github.com/satori/go.uuid"
)

const serverNodePrefix = "Server:"

// Server implementation
type Server struct {
	cfg          *Config              // Server configuration
	ctx          context.Context      // Context for the service
	cancel       context.CancelFunc   // Cancellation function
	wg           sync.WaitGroup       // Wait group
	id           string               // Unique server id
	http         *http.Server         // Underlying HTTP server
	hub          *Hub                 // Messaging hub
	messageQueue *MessageQueue        // Message queue service
	registrar    *registrar.Registrar // Node registrar service
}

type handler func(*Server, http.ResponseWriter, *http.Request) error

func newServer(cfg *Config) (s *Server) {
	ctx, cancel := context.WithCancel(context.Background())
	id := serverNodePrefix + uuid.NewV4().String()

	log.Printf("[Info] Server ID: %s", id)
	log.Printf("[Info] Starting server on port %s", cfg.port)

	s = &Server{
		cfg:          cfg,
		ctx:          ctx,
		cancel:       cancel,
		id:           id,
		hub:          NewHub(),
		messageQueue: NewMessageQueue(id, cfg.redisQueueService),
		registrar: registrar.New(
			id,
			cfg.externalIP,
			cfg.port,
			cfg.redisNodeRegistryService,
		),
	}

	r := mux.NewRouter()
	r.HandleFunc("/ws", s.wrapMiddleware(serveWs)).Methods("GET")

	s.http = &http.Server{
		Handler: r,
		Addr:    ":" + cfg.port,
	}
	return
}

func (s *Server) start() (err error) {
	cleanup.Listen(s.cancel, &s.wg)

	s.hub.Start()

	if err = s.messageQueue.Start(); err != nil {
		return
	}

	if err = s.registrar.Start(); err != nil {
		return
	}

	cleanup.Add(s.ctx, &s.wg, s.messageQueue.Stop)
	cleanup.Add(s.ctx, &s.wg, s.registrar.Stop)
	cleanup.Add(s.ctx, &s.wg, s.hub.Stop)

	err = s.http.ListenAndServe()
	return
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
