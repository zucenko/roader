package model

import log "github.com/sirupsen/logrus"

func (cell *Cell) Crossings() {
	cell.Crossing = false
	var pl1 *Player
	for _, p := range cell.Paths {
		if p.Player != nil {
			if pl1 == nil {
				pl1 = p.Player
			} else if pl1 != p.Player {
				cell.Crossing = true
				return
			}
		}
	}
}

func NewEmptyModel(cols, rows int, players map[int32]Player) *Model {
	matrix := make([][]*Cell, 0)
	// create
	for c := 0; c < cols; c++ {
		column := make([]*Cell, 0)
		for r := 0; r < rows; r++ {
			column = append(column, &Cell{Col: c, Row: r})
		}
		matrix = append(matrix, column)
	}
	// connect
	for c := 0; c < cols; c++ {
		for r := 0; r < rows; r++ {
			cell := matrix[c][r]
			if c == 0 {
				cell.Paths[2] = &Path{Dir: 2}
			} else {
				// connect prev left
				prevCell := matrix[c-1][r]
				prevCell.Paths[0].Target = cell
				cell.Paths[2] = &Path{Dir: 2, Target: prevCell}
			}
			if r == 0 {
				cell.Paths[3] = &Path{Dir: 3}
			} else {
				prevCell := matrix[c][r-1]
				prevCell.Paths[1].Target = cell
				cell.Paths[3] = &Path{Dir: 3, Target: prevCell}
			}
			cell.Paths[0] = &Path{Dir: 0}
			cell.Paths[1] = &Path{Dir: 1}
		}
	}

	p := make(map[int32]*Player)

	for id, pl := range players {
		pp := players[id]
		p[id] = &pp
		matrix[pp.Col][pp.Row].Player = &pp
		log.Printf("%v", pl)
	}

	log.Printf("playerMap: %v", p)

	return &Model{
		Matrix:  matrix,
		Players: p,
	}
}

func UnsuccesMoveMessage(d int, p *Player) ServerMessage {
	return ServerMessage{
		Directions: []DirectionSuccess{{
			Direction: d,
			Col:       p.Col,
			Row:       p.Row,

			Success: false}},
	}
}
