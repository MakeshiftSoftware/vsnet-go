package node

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
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
	return n.redis.Ping()
}

// getMinionsHandler http handler to get all minions
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

// getMinionHandler http handler to get minion by id
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

// sendMessageHandler http handler to send message to a minion
func sendMessageHandler(n *Node, w http.ResponseWriter, r *http.Request) error {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		return err
	}

	return n.sendMessage(mux.Vars(r)["id"], b)
}

// broadcastHandler http handler to broadcast message to all minions
func broadcastHandler(n *Node, w http.ResponseWriter, r *http.Request) error {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		return err
	}

	return n.broadcast(b)
}
