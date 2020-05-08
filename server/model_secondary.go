package server

import (
	"fmt"
	"github.com/gorilla/websocket"
)

const HTTP_SUCCESS = 200
const HTTP_BAD_REQUEST = 400
const HTTP_NOT_FOUND = 404
const HTTP_TIMEOUT = 408
const HTTP_SERVER_ERR = 503

type ResponseCode int

const (
	GAME_READY ResponseCode = iota
	GAME_NOT_FOUND
	GAME_INVALIDE
)

func (h ResponseCode) ToHttp() int {
	switch h {
	case GAME_READY:
		return HTTP_SUCCESS
	case GAME_NOT_FOUND:
		return HTTP_NOT_FOUND
	case GAME_INVALIDE:
		return HTTP_BAD_REQUEST
	default:
		panic(h)
	}
}

func (gss GameSessionState) Name() string {
	switch gss {
	case GS_NEW:
		return "GS_NEW"
	case GS_PLAY:
		return "GS_PLAY"
	case GS_OVER:
		return "GS_OVER"
	default:
		return fmt.Sprintf("n/a:%d", gss)
	}
}

func (ps PlayerSessionState) Name() string {
	switch ps {
	case PS_NEW:
		return "NEW"
	case PS_PLAY:
		return "PLAY"
	case PS_ERR:
		return "ERR"
	case PS_ERR_SEC:
		return "ERR_SEC"
	default:
		return "N/A"
	}
}

type GameContextAwaiting struct {
	ResponseCode ResponseCode
	GameSession  *GameSession
}

type GameRequest struct {
	GameContextAwaiting chan GameContextAwaiting
}

type PlayerConnectRequest struct {
	Con      *websocket.Conn
	GameOver chan struct{}
}

type PlayerEvent struct {
	Player    int32
	GameEvent GameEvent
}

type GameEvent struct {
	Direction int
}
