package main

type Player struct {
	Color    GameColor
	col, row int
}

type Portal struct {
	Target *Cell
}

type Path struct {
	Target *Cell
	Wall   bool
	Player Player
}

type Cell struct {
	col, row int
	Paths    [4]*Path
	Portal   Portal
	Diamond  bool
}

type Model struct {
	Matrix  [][]Cell
	Players []Player
}
