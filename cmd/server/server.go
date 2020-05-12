package main

import (
	"github.com/matryer/way"
	log "github.com/sirupsen/logrus"
	"github.com/zucenko/roader/server"
	"net/http"
	"os"
)

type Server struct {
	router     *way.Router
	GameServer *server.GameServer
}

func main() {
	// vyrobit router a http server
	// spustit handlovani pripojovani
	Server := Server{
		GameServer: server.NewGameServer(),
	}
	go Server.GameServer.Loop()
	Server.routes()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	log.Fatalln(http.ListenAndServe(":"+port, Server.router))
}
