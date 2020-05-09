package model

type Player struct {
	Id       int32
	Keys     int
	Diamonds int
	Col, Row int
}

type Portal struct {
	Target *Cell
}

type Path struct {
	Dir    int
	Target *Cell
	Wall   bool
	Lock   bool
	Player *Player
}

type Cell struct {
	Col, Row int
	Paths    [4]*Path
	Portal   *Portal
	Player   *Player
	Diamond  bool
	Key      bool
}

type Model struct {
	Matrix     [][]*Cell
	Players    map[int32]*Player
	PlayerKeys []int32
}
