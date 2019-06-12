package node

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// handler represents a custom http route handler function.
type handler func(*Node, http.ResponseWriter, *http.Request) error

// wrapMiddleware wraps a custom http handler function and returns a handler function
// in the format that is expected by the http server.
func (n *Node) wrapMiddleware(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(n, w, r)

		if err != nil {
			log.Printf("[error] %+v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// healthcheckHandler is an http handler function that performs a node healthcheck.
func healthcheckHandler(n *Node, w http.ResponseWriter, r *http.Request) error {
	return n.redis.Ping()
}

// getMinionsHandler is an http handler function that retrieves all active minions.
func getMinionsHandler(n *Node, w http.ResponseWriter, r *http.Request) error {
	var minions []Minion
	var res []byte
	var err error

	if minions, err = n.getMinions(); err != nil {
		return err
	}

	if res, err = json.Marshal(minions); err != nil {
		return err
	}

	_, err = w.Write(res)

	return err
}

// getMinionHandler is an http handler function that retrieves a minion by its id.
func getMinionHandler(n *Node, w http.ResponseWriter, r *http.Request) error {
	var minion Minion
	var res []byte
	var err error

	if minion, err = n.getMinion(mux.Vars(r)["id"]); err != nil {
		return err
	}

	if res, err = json.Marshal(minion); err != nil {
		return err
	}

	_, err = w.Write(res)

	return err
}

// sendMessageHandler is an http handler function that sends a message to a specific minion by its id.
func sendMessageHandler(n *Node, w http.ResponseWriter, r *http.Request) error {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		return err
	}

	return n.sendMessage(mux.Vars(r)["id"], b)
}

// broadcastMessageHandler is an http handler function that broadcasts a message to all active minions.
func broadcastMessageHandler(n *Node, w http.ResponseWriter, r *http.Request) error {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		return err
	}

	return n.broadcastMessage(b)
}
