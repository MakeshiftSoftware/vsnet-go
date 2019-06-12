package node

import (
	"log"
	"net/http"
)

// handler represents a custom http route handler function.
type handler func(*node, http.ResponseWriter, *http.Request) error

// wrapMiddleware wraps a custom http handler function and returns a handler function
// in the format that is expected by the http server.
func (n *node) wrapMiddleware(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(n, w, r)

		if err != nil {
			log.Printf("[error] %+v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// healthcheckHandler is an http handler function that performs a node healthcheck.
func healthcheckHandler(n *node, w http.ResponseWriter, r *http.Request) error {
	ok, err := n.redis.Exists(nodePrefix + n.id)

	if err != nil {
		return err
	}

	if !ok {
		return ErrMinionNotFound
	}

	return nil
}

// serveWs is an http handler function that upgrades websocket connection requests.
func serveWs(n *node, w http.ResponseWriter, r *http.Request) error {
	sock, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		return err
	}

	return n.hub.onClientConnected(sock)
}
