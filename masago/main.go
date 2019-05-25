package main

import (
	"log"

	"github.com/makeshiftsoftware/vsnet/masago/config"
	"github.com/makeshiftsoftware/vsnet/masago/messenger"
)

func main() {
	cfg := config.New()
	svc := messenger.NewService(cfg)

	if err := svc.Start(); err != nil {
		svc.Cancel()
		log.Fatalf("[error] %+v", err)
	}
}
