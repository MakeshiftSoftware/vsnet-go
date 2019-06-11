package main

import (
	"log"

	"github.com/makeshiftsoftware/vsnet/master/internal/config"
	"github.com/makeshiftsoftware/vsnet/master/internal/node"
)

func main() {
	n := node.New(config.New())

	if err := n.Start(); err != nil {
		n.Cleanup()
		log.Fatalf("[error] %+v", err)
	}
}
