package messenger

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/makeshiftsoftware/vsnet/masago/config"
	"github.com/makeshiftsoftware/vsnet/pkg/cleanup"
	"github.com/makeshiftsoftware/vsnet/pkg/registrar"
	uuid "github.com/satori/go.uuid"
)

// Service implementation
type Service struct {
	context   context.Context      // Context for the service
	cancel    context.CancelFunc   // Cancellation function
	wg        sync.WaitGroup       // Wait group
	Config    *config.Config       // Server configuration
	name      string               // Unique server name
	http      *http.Server         // Underlying HTTP server
	hub       *Hub                 // Hub service
	registrar *registrar.Registrar // Registrar service
}

type handler func(*Service, http.ResponseWriter, *http.Request) error

// NewService creates a new service
func NewService(cfg *config.Config) (s *Service) {
	ctx, cancel := context.WithCancel(context.Background())
	name := cfg.RedisServerPrefix + uuid.NewV4().String()

	hubService := NewHub(name, cfg.RedisBrokerAddr)
	registrarService := registrar.New(name, cfg.ExternalIP, cfg.Port, cfg.RedisRegistrarAddr)

	s = &Service{
		context:   ctx,
		cancel:    cancel,
		Config:    cfg,
		name:      name,
		hub:       hubService,
		registrar: registrarService,
	}

	r := mux.NewRouter()
	r.HandleFunc("/healthz", s.wrapMiddleware(healthcheck)).Methods("GET")
	r.HandleFunc("/ws", s.wrapMiddleware(serveWs)).Methods("GET")

	s.http = &http.Server{
		Handler: r,
		Addr:    ":" + cfg.Port,
	}
	return
}

// Start starts the service
func (s *Service) Start() error {
	log.Printf("[info] starting service")
	cleanup.Listen(s.cancel, &s.wg)
	cleanup.Add(s.context, &s.wg, s.hub.Stop)
	cleanup.Add(s.context, &s.wg, s.registrar.Stop)

	if err := s.registrar.Start(); err != nil {
		return err
	}

	if err := s.hub.Start(); err != nil {
		return err
	}

	return s.http.ListenAndServe()
}

// Cancel cancel service context and wait for cleanup
func (s *Service) Cancel() {
	log.Printf("[info] stopping service")
	s.cancel()
	s.wg.Wait()
}

func healthcheck(s *Service, w http.ResponseWriter, r *http.Request) error {
	if err := s.registrar.Ready(); err != nil {
		return err
	}
	if err := s.hub.Ready(); err != nil {
		return err
	}
	return nil
}

func (s *Service) wrapMiddleware(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(s, w, r)

		if err != nil {
			log.Printf("[Error] %+v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
