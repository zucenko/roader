package main

import (
	"github.com/matryer/way"
)

const URI_WS = "/play"

func (s *Server) routes() {
	s.router = way.NewRouter()
	s.router.HandleFunc("GET", URI_WS, s.GameServer.HandleHttpCall())
}
