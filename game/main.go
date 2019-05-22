package main

import (
	"log"
)

func main() {
	cfg := loadConfig()

	s := newServer(cfg)

	if err := s.start(); err != nil {
		log.Fatalf("[Error] %+v", err)
	}
}
