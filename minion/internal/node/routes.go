package node

import (
	"log"
	"net/http"
)

// Custom http handler func
type handler func(*Node, http.ResponseWriter, *http.Request) error

// wrapMiddleware wraps http handler func
func (n *Node) wrapMiddleware(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(n, w, r)

		if err != nil {
			log.Printf("[error] %+v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// healthcheckHandler http handler to perform node healthcheck
func healthcheckHandler(n *Node, w http.ResponseWriter, r *http.Request) error {
	ok, err := n.redis.Exists(nodePrefix + n.id)

	if err != nil {
		return err
	}

	if !ok {
		return ErrMinionNotFound
	}

	return nil
}

// serveWs http handler to upgrade websocket connection requests
func serveWs(n *Node, w http.ResponseWriter, r *http.Request) error {
	sock, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		return err
	}

	return n.hub.onClientConnected(sock)
}
