package server

import (
	"github.com/gorilla/websocket"
	"github.com/zucenko/roader/model"
	"time"
)

type GameServer struct {
	GameSessions []*GameSession
	GameRequests chan GameRequest
	Upgrader     *websocket.Upgrader
}

type GameSessionState int

const (
	GS_NEW GameSessionState = iota
	GS_WAIT
	GS_PLAY
	GS_ERR
	GS_OVER
)

type GameSession struct {
	State                 GameSessionState
	Model                 *model.Model
	PlayerSessions        []PlayerSession
	Errors                chan int32
	Events                chan PlayerEvent
	PlayerConnectRequests chan PlayerConnectRequest
}

type PlayerSessionState int

const (
	PS_NEW PlayerSessionState = iota + 1
	PS_PLAY
	PS_OVER
	PS_ERR
	PS_ERR_SEC
)

type PlayerSession struct {
	State       PlayerSessionState
	Id          int32
	GameSession *GameSession
	Conn        *websocket.Conn
	GameOver    chan struct{}

	MessagesToSend chan model.ServerMessage

	DebugInMessages  int
	DebugOutMessages int
	DebugLastMessage time.Time
	DebugLastPing    time.Time
	DebugPings       int
}

//
