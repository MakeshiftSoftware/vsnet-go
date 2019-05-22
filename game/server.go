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
	ctx          context.Context      // The context for the service
	cancel       context.CancelFunc   // The cancellation function
	wg           sync.WaitGroup       // The wait group
	cfg          *Config              // The server configuration
	id           string               // The unique server id
	srv          *http.Server         // The underlying HTTP server
	messageQueue *MessageQueue        // The message queue service
	registrar    *registrar.Registrar // The node registrar service
}

type handler func(*Server, http.ResponseWriter, *http.Request) error

func newServer(cfg *Config) (s *Server) {
	ctx, cancel := context.WithCancel(context.Background())
	id := serverNodePrefix + uuid.NewV4().String()

	log.Printf("[Info] Server ID: %s", id)
	log.Printf("[Info] Starting server on port %s", cfg.port)

	s = &Server{
		ctx:          ctx,
		cancel:       cancel,
		cfg:          cfg,
		id:           id,
		messageQueue: NewMessageQueue(id, cfg.redisQueueService),
		registrar: registrar.New(
			id,
			cfg.externalIP,
			cfg.port,
			cfg.redisNodeRegistryService,
		),
	}

	r := s.newHandler()

	s.srv = &http.Server{
		Handler: r,
		Addr:    ":" + cfg.port,
	}
	return
}

func (s *Server) start() (err error) {
	cleanup.Listen(s.cancel, &s.wg)

	if err = s.messageQueue.Start(s.ctx, &s.wg); err != nil {
		return
	}

	if err = s.registrar.Start(s.ctx, &s.wg); err != nil {
		return
	}

	cleanup.Add(s.ctx, &s.wg, s.messageQueue.Stop)
	cleanup.Add(s.ctx, &s.wg, s.registrar.Stop)

	err = s.srv.ListenAndServe()
	return
}

func (s *Server) newHandler() (h http.Handler) {
	h = mux.NewRouter()
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
