package main

import (
	"net/http"

	"github.com/makeshiftsoftware/vsnet/pkg/auth"
	"github.com/pkg/errors"
)

func newConn(s *Server, w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query()
	token := query.Get("token")
	gameID := query.Get("game_id")

	if token == "" {
		return ErrMissingToken
	}

	if gameID == "" {
		return ErrMissingGameID
	}

	key, err := auth.NewAccessKey(token, &cfg.jwtSecret)

	if err != nil {
		return err
	}

	con, err := cfg.upgrader.Upgrade(w, r, nil)

	if err != nil {
		return errors.Wrap(err, "Error upgrading connection")
	}

	c := newClient(key, s.hub, con)
	c.hub.register <- c

	go c.writePump()
	go c.readPump()

	return nil
}

func readinessCheck(s *Server, w http.ResponseWriter, r *http.Request) error {
	if err := s.gameState.Ping(); err != nil {
		return errors.Wrap(err, "Game state redis ping failed")
	} else if err := s.presence.Ping(); err != nil {
		return errors.Wrap(err, "Presence redis ping failed")
	} else if err := s.pubsub.Ping(); err != nil {
		return errors.Wrap(err, "Pubsub redis ping failed")
	} else if err := s.nodeRegistry.Ping(); err != nil {
		return errors.Wrap(err, "Node registry redis ping failed")
	}

	_, err := s.getNode(s.id)

	return errors.Wrap(err, "Readiness check failed")
}
