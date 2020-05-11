package server

import (
	"encoding/gob"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/zucenko/roader/model"
	"net"
	"net/http"
	"time"
)

func NewGameServer() *GameServer {
	return &GameServer{
		GameSessions: make([]*GameSession, 0),
		GameRequests: make(chan GameRequest, 0),
		Upgrader:     &websocket.Upgrader{},
	}
}

func (s *GameServer) HandleHttpCall() http.HandlerFunc {
	timeout := 200 * time.Millisecond
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("HandleHttpCall - Conection received.............................")

		gcas := make(chan GameContextAwaiting)
		select {
		case s.GameRequests <- GameRequest{GameContextAwaiting: gcas}:
			log.Printf("HandleHttpCall -> MessagingServer.GameConnectRequests")
		case <-time.After(timeout):
			log.Warn("GameRequests TIMEOUTED")
			w.WriteHeader(HTTP_TIMEOUT)
			return
		}

		// find/create to GameContext
		var gca GameContextAwaiting
		select {
		case gca = <-gcas:
			log.Printf("HandleHttpCall GameContextAwaiting <- code:%d", gca.ResponseCode)
			switch gca.ResponseCode {
			case GAME_NOT_FOUND:
				fallthrough
			case GAME_INVALIDE:
				w.WriteHeader(gca.ResponseCode.ToHttp())
				return
			case GAME_READY: //ony good option
				log.Printf("HandleHttpCall ok, have GameSession")

			default:
				log.Errorf("gca.ResponseCode not expected:%v", gca.ResponseCode)
			}
		case <-time.After(timeout):
			log.Warnf("HandleHttpCall GameContextAwaiting <- TIMEOUTED")
			// SERVICE_UNAVAILABLE
			w.WriteHeader(HTTP_TIMEOUT)
			return
		}

		log.Info("HandleHttpCall lets upgrade websocket ")
		// upgrade
		con, err := s.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("HandleHttpCall websocket upgrade err %v", err)
			w.WriteHeader(HTTP_SERVER_ERR)
			return
		}

		defer con.Close()

		// gameover callback channel
		gameOver := make(chan struct{})
		// send message to MessagingServer GameConnectRequests and register GameOver

		log.Info("HandleHttpCall request ")
		select {
		case gca.GameSession.PlayerConnectRequests <- PlayerConnectRequest{
			Con:      con,
			GameOver: gameOver}:
		case <-time.After(timeout):
			w.WriteHeader(HTTP_TIMEOUT)
			return
		}

		// wait till game over
		log.Info("HandleHttpCall and wait for gameover ")
		<-gameOver
	}
}

func (s *GameServer) Loop() {
	log.Printf("GameServer.Loop starting")
	for {
		select {
		case gameReq := <-s.GameRequests:
			log.Printf("GameServer.Loop gameReq")
			// najdu session nebo udelam novou
			newIndex := -1
			for i, gs := range s.GameSessions {
				if gs.State == GS_WAIT {
					newIndex = i
					break
				}
			}
			var gs *GameSession
			if newIndex == -1 {
				log.Info("create GameSession")
				model, err := Load()
				if err != nil {
					log.Printf("ERR LOADING %v", err)
				}
				gs = &GameSession{
					State:                 GS_NEW,
					Model:                 model,
					PlayerSessions:        make([]PlayerSession, 0),
					Errors:                make(chan int32),
					Events:                make(chan PlayerEvent),
					PlayerConnectRequests: make(chan PlayerConnectRequest),
				}
				go gs.Loop()
				s.GameSessions = append(s.GameSessions, gs)
			} else {
				gs = s.GameSessions[newIndex]
			}

			//let know it is success
			gameReq.GameContextAwaiting <- GameContextAwaiting{
				ResponseCode: GAME_READY,
				GameSession:  gs,
			}
		}
	}
}

func (gs *GameSession) Loop() {
	log.Info("GameSession.Loop start")
loop:
	for {
		select {
		case pcr := <-gs.PlayerConnectRequests:
			log.Info("GameSession.Loop PlayerConnectRequests")
			gs.addPlayer(
				pcr.Con,
				pcr.GameOver)
			if gs.State == GS_NEW && len(gs.PlayerSessions) < len(gs.Model.PlayerKeys) {
				log.Printf("GameSession.Loop not enought players")
				gs.State = GS_WAIT
			} else {
				gs.State = GS_PLAY
				for i, ps := range gs.PlayerSessions {
					gs.PlayerSessions[i].State = PS_PLAY
					ps.MessagesToSend <- ps.MakeGameSetupMessage()
				}
			}
		case errPlayer := <-gs.Errors:
			log.Warn("killing GS")
			gs.State = GS_ERR
			for i, ps := range gs.PlayerSessions {
				if ps.Id == errPlayer {
					gs.PlayerSessions[i].State = PS_ERR
				} else {
					gs.PlayerSessions[i].State = PS_ERR_SEC
				}
			}
		case pe := <-gs.Events:
			// tady je realna hra!!!!
			var playerSession, opponentSession *PlayerSession
			for i, ps := range gs.PlayerSessions {
				if ps.Id == pe.PlayerId {
					playerSession = &gs.PlayerSessions[i]
				} else {
					opponentSession = &gs.PlayerSessions[i]
				}

			}

			var delay int64
			switch gs.TurnType(pe) {
			case MP_NOT_POSSIBLE:
				player := gs.Model.Players[playerSession.Id]
				playerSession.MessagesToSend <- model.UnsuccesMoveMessage(pe.GameEvent.Direction, player)
				continue loop
			case MP_MINE:
				delay = 0
			case MP_TELEPORT:
				delay = 1000
			case MP_VIRGIN:
				delay = 150
			case MP_OPPONENT:
				delay = 2000
			}
			reply := func() {
				timer1 := time.NewTimer(time.Duration(delay) * time.Millisecond)
				<-timer1.C
				messageToPlayer, messageToOpponent := gs.Turn(pe)
				if messageToPlayer != nil {
					//log.Info("GameSession.Loop messageToPlayer")
					playerSession.MessagesToSend <- *messageToPlayer
				}
				if messageToOpponent != nil {
					if opponentSession != nil {
						//log.Info("GameSession.Loop messageToOpponent")
						opponentSession.MessagesToSend <- *messageToOpponent
					}
				}
			}
			go reply()
		}
	}
}

type MovePossible int

const (
	MP_NOT_POSSIBLE MovePossible = iota
	MP_MINE
	MP_VIRGIN
	MP_TELEPORT
	MP_OPPONENT
)

func (gs *GameSession) TurnType(pe PlayerEvent) MovePossible {
	player := gs.Model.Players[pe.PlayerId]
	cell := gs.Model.Matrix[player.Col][player.Row]

	if pe.GameEvent.Direction == 4 {
		if cell.Portal == nil || cell.Portal.Target.Player != nil {
			return MP_NOT_POSSIBLE
		}
		return MP_TELEPORT
	} else {

		path := cell.Paths[pe.GameEvent.Direction]

		//unsuccess
		if path.Target == nil ||
			path.Wall && !path.Lock ||
			path.Wall && path.Lock && player.Keys == 0 ||
			path.Target.Player != nil {
			return MP_NOT_POSSIBLE
		}
		// mine
		if path.Player == player {
			return MP_MINE
		}
		// virgin
		if path.Player == nil {
			return MP_VIRGIN
		}
		// opponent
		return MP_OPPONENT
	}
}

func (gs *GameSession) Turn(pe PlayerEvent) (
	messageToPlayer *model.ServerMessage,
	messageToOpponent *model.ServerMessage) {
	c, r, k, d, suc := move(gs.Model, pe.PlayerId, pe.GameEvent.Direction)
	directionSuccess := []model.DirectionSuccess{{
		Direction: pe.GameEvent.Direction,
		Col:       c, Row: r, Success: suc,
		PlayerKey: pe.PlayerId}}
	if suc {
		player := gs.Model.Players[pe.PlayerId]
		messageToPlayer = &model.ServerMessage{}
		messageToOpponent = &model.ServerMessage{}
		messageToPlayer.Directions = directionSuccess
		messageToOpponent.Directions = directionSuccess
		infos := info(gs.Model.Matrix[c][r], true)
		for i, p := range gs.Model.Matrix[c][r].Paths {
			if p != nil && !p.Wall && p.Target != nil {
				infos = append(infos, info(p.Target, true)...)
			}
			diagonal := diagonal(gs.Model.Matrix[c][r], i)
			if diagonal != nil {
				infos = append(infos, info(diagonal, true)...)
			}
			faar := farDirect(gs.Model.Matrix[c][r], i)
			if faar != nil {
				infos = append(infos, info(faar, true)...)
			}
		}
		messageToPlayer.Visibles = infos
		if k != 0 || d != 0 {
			messageToPlayer.Picks = []model.Pick{{Keys: player.Keys, Diamonds: player.Diamonds}}
		}
	} else {
		messageToPlayer = &model.ServerMessage{}
		messageToPlayer.Directions = directionSuccess
	}
	return
}

func (gs *GameSession) addPlayer(
	conn *websocket.Conn,
	gameOver chan struct{},
) {
	log.Printf("GameSession.addPlayer")
	playerId := gs.Model.PlayerKeys[len(gs.PlayerSessions)]
	ps := PlayerSession{
		State:          PS_NEW,
		Id:             playerId,
		GameSession:    gs,
		Conn:           conn,
		GameOver:       gameOver,
		MessagesToSend: make(chan model.ServerMessage, 10),
	}
	conn.SetPingHandler(
		func(message string) error {
			err := conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(time.Second))
			ps.DebugLastPing = time.Now()
			ps.DebugPings++
			if err == websocket.ErrCloseSent {
				return nil
			} else if e, ok := err.(net.Error); ok && e.Temporary() {
				return nil
			}
			return err
		})
	// start processing input form PGS
	go ps.LoopChannelRead()
	// start sending from server
	go ps.LoopChannelWrite()
	// add to game session
	gs.PlayerSessions = append(gs.PlayerSessions, ps)
}

func (ps *PlayerSession) LoopChannelRead() {
	log.Printf("LoopChannelRead STARTED")
loop:
	for {
		_, r, err := ps.Conn.NextReader()
		if err != nil {
			if ps.State == PS_ERR_SEC {
				log.Printf("LoopChannelRead made by close THE OTHER ONE")
			} else {
				log.Printf("LoopChannelRead err reading message from Conn %v", err)
				ps.State = PS_ERR
				ps.GameSession.Errors <- ps.Id
			}
			break loop
		}
		//log.Printf("LoopChannelRead received  message type: %d", messageType)
		dec := gob.NewDecoder(r)
		cm := &model.ClientMessage{}
		err = dec.Decode(cm)
		if err != nil {
			log.Warn("cant decode")
			ps.State = PS_ERR
			ps.GameSession.Errors <- ps.Id
			break loop
		}
		ps.DebugLastMessage = time.Now()
		ps.DebugInMessages++
		select {
		case ps.GameSession.Events <- PlayerEvent{
			PlayerId:  ps.Id,
			GameEvent: GameEvent{Direction: cm.Move},
		}:
		default:
			log.Warnf("Dropping Data red from socket but.. GameContext.Events FULL")
		}
	}
	log.Printf("LoopChannelRead ENDED")
}

func (ps *PlayerSession) MakeGameSetupMessage() model.ServerMessage {
	var Visibles []model.Visibilize

	Players := make(map[int32]model.Player)
	for id, pl := range ps.GameSession.Model.Players {
		Players[id] = *pl
		if pl.Id == ps.Id {
			p := ps.GameSession.Model.Players[ps.Id]
			Visibles = info(ps.GameSession.Model.Matrix[p.Col][p.Row], true)
		}
	}
	return model.ServerMessage{
		Setup: []model.Setup{{
			Cols:      len(ps.GameSession.Model.Matrix),
			Rows:      len(ps.GameSession.Model.Matrix[0]),
			PlayerKey: ps.Id,
			Players:   Players,
		}},
		Directions: []model.DirectionSuccess{},
		Visibles:   Visibles,
	}

}

// this function only consumes. no worries about full buffer stuck
func (ps *PlayerSession) LoopChannelWrite() {
	log.Printf("PlayerSession.LoopChannelWrite STARTED")
loop:
	for {
		select {
		case mes := <-ps.MessagesToSend:
			//log.Printf("PlayerSession.LoopChannelWrite started key:%v", ps.Id)
			if ps.State == PS_ERR || ps.State == PS_ERR_SEC {
				log.Printf("LoopChannelWrite it was close event")
				break loop
			}
			//log.Printf("PlayerSession.LoopChannelWrite WRITE TO WEBSOCKET >>>   ")
			w, err := ps.Conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				log.Warn("PlayerSession.LoopChannelWrite cant get writer %v", err)
				ps.State = PS_ERR
				ps.GameSession.Errors <- ps.Id
				break loop
			}
			enc := gob.NewEncoder(w)
			err = enc.Encode(mes)
			if err != nil {
				log.Warn("PlayerSession.LoopChannelWrite cant encode %v", err)
				ps.State = PS_ERR
				ps.GameSession.Errors <- ps.Id
				break loop
			}
			err = w.Close()
			if err != nil {
				log.Warn("PlayerSession.LoopChannelWrite cant  encode %v", err)
				ps.State = PS_ERR
				ps.GameSession.Errors <- ps.Id
				break loop
			}
			ps.DebugOutMessages++
		}
	}
	log.Printf("LoopChannelWrite ENDED")
}

func move(m *model.Model, playerId int32, d int) (col int, row int, keysAdd int, diamondsAdd int, success bool) {
	player, _ := m.Players[playerId]
	cell := m.Matrix[player.Col][player.Row]
	col = player.Col
	row = player.Row
	if d == 4 {
		if cell.Portal != nil && cell.Portal.Target.Player == nil {
			newCell := cell.Portal.Target
			newCell.Player = cell.Player
			newCell.Player.Col = newCell.Col
			newCell.Player.Row = newCell.Row
			//newCell.Unhook(playerId)
			cell.Player = nil
			col = player.Col
			row = player.Row
			success = true
		}
		return
	}

	if cell.Paths[d] == nil || cell.Paths[d].Target != nil && cell.Paths[d].Target.Player != nil {
		return player.Col, player.Row, 0, 0, false
	}

	if !cell.Paths[d].Wall || cell.Paths[d].Lock && player.Keys > 0 {

		newCell := cell.Paths[d].Target
		//newCell.Unhook(playerId)

		if cell.Paths[d].Lock {
			player.Keys--
			keysAdd--
			cell.Paths[d].Wall = false
			cell.Paths[d].Lock = false
			cell.Paths[d].Target.Paths[(d+2)%4].Wall = false
			cell.Paths[d].Target.Paths[(d+2)%4].Lock = false
		}

		if cell.Paths[d].Target != nil {
			cell.Paths[d].Player = cell.Player
			newCell.Player = cell.Player
			newCell.Player.Col = newCell.Col
			newCell.Player.Row = newCell.Row
			newCell.Paths[(d+2)%4].Player = newCell.Player
			cell.Player = nil

			if newCell.Diamond {
				newCell.Diamond = false
				player.Diamonds++
				diamondsAdd = 1
			}
			if newCell.Key {
				newCell.Key = false
				player.Keys++
				keysAdd = 1
			}
		}
		col = player.Col
		row = player.Row
		success = true
		return
	}
	col = player.Col
	row = player.Row
	return
}

func info(cell *model.Cell, wallsToo bool) []model.Visibilize {
	var walls []bool
	var locks []bool
	basic := visibilizerFromCell(cell)
	if wallsToo {
		for _, p := range cell.Paths {
			if p != nil {
				walls = append(walls, p.Wall)
				locks = append(locks, p.Lock)
			}
		}
		basic.Walls = walls
		basic.Locks = locks
	}
	visibles := []model.Visibilize{basic}
	if cell.Portal != nil {
		portalTarget := visibilizerFromCell(cell)
		visibles = append(visibles, portalTarget)
	}
	return visibles
}

func visibilizerFromCell(cell *model.Cell) model.Visibilize {
	portal := false
	portalCol, portalRow := 0, 0
	if cell == nil {
		portalCol = 112
	}
	if cell.Portal != nil {
		portal = true
		portalCol, portalRow = cell.Portal.Target.Col, cell.Portal.Target.Row
	}
	var pid int32
	var hasPlayer bool
	if cell.Player != nil {
		pid = cell.Player.Id
		hasPlayer = true
	}
	return model.Visibilize{
		PlayerId:    pid,
		HasPlayer:   hasPlayer,
		Col:         cell.Col,
		Row:         cell.Row,
		Diamond:     cell.Diamond,
		Key:         cell.Key,
		Portal:      portal,
		PortalToCol: portalCol,
		PortalToRow: portalRow,
	}
}

func diagonal(c *model.Cell, dirPlus int) *model.Cell {
	if c.Paths[dirPlus].Wall || c.Paths[(dirPlus+1)%4].Wall {
		return nil
	}
	if c.Paths[dirPlus].Target == nil || c.Paths[(dirPlus+1)%4].Target == nil {
		return nil
	}

	if c.Paths[dirPlus].Target == nil && c.Paths[(dirPlus+1)%4].Wall {
		return nil
	}

	if c.Paths[(dirPlus+1)%4].Target == nil && c.Paths[dirPlus].Wall {
		return nil
	}

	if c.Paths[dirPlus].Wall && c.Paths[(dirPlus+1)%4].Wall {
		return nil
	}
	switch dirPlus {
	case 0:
		if c.Paths[0].Target.Paths[1].Wall && c.Paths[1].Target.Paths[0].Wall {
			return nil
		}
		return c.Paths[0].Target.Paths[1].Target
	case 1:
		if c.Paths[1].Target.Paths[2].Wall && c.Paths[2].Target.Paths[1].Wall {
			return nil
		}
		return c.Paths[1].Target.Paths[2].Target
	case 2:
		if c.Paths[2].Target.Paths[3].Wall && c.Paths[3].Target.Paths[2].Wall {
			return nil
		}
		return c.Paths[2].Target.Paths[3].Target
	case 3:
		if c.Paths[3].Target.Paths[0].Wall && c.Paths[0].Target.Paths[3].Wall {
			return nil
		}
		return c.Paths[3].Target.Paths[0].Target
	}
	return nil
}

func farDirect(c *model.Cell, dirPlus int) *model.Cell {
	if c.Paths[dirPlus].Wall {
		return nil
	}
	if c.Paths[dirPlus].Target == nil || c.Paths[dirPlus].Target.Paths[dirPlus] == nil {
		return nil
	}
	if c.Paths[dirPlus].Target.Paths[dirPlus].Wall {
		return nil
	}
	if c.Paths[dirPlus].Target.Paths[dirPlus].Target == nil {
		return nil
	}
	return c.Paths[dirPlus].Target.Paths[dirPlus].Target
}
