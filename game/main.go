package main

import (
	"log"
)

func main() {
	cfg := loadConfig()

	s := newServer(cfg)

	if err := s.start(); err != nil {
		if s.cancel != nil {
			s.cancel()
		}
		s.wg.Wait()
		log.Fatalf("[Error] %+v", err)
	}
}
