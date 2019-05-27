package main

import (
	"log"

	"github.com/makeshiftsoftware/vsnet/masago/config"
	"github.com/makeshiftsoftware/vsnet/masago/messenger"
)

func main() {
	cfg := config.New()
	s := messenger.NewService(cfg)

	if err := s.Start(); err != nil {
		s.Cancel()
		log.Fatalf("[error] %+v", err)
	}
}
