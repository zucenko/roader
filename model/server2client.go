package model

type ServerMessage struct {
	Setup      []Setup
	Directions []DirectionSuccess
	Visibles   []Visibilize
	Picks      []Pick
}

type Setup struct {
	Cols, Rows int
	PlayerKey  int32
	Players    map[int32]Player
}

type DirectionSuccess struct {
	Direction int
	Col, Row  int
	Success   bool
	PlayerKey int32
}

type Visibilize struct {
	Col, Row    int
	Walls       []bool
	Locks       []bool
	Diamond     bool
	Key         bool
	Portal      bool
	PortalToCol int
	PortalToRow int
	HasPlayer   bool
	PlayerId    int32
}

type Pick struct {
	Keys     int
	Diamonds int
}
