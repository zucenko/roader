package model

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
		p[id] = &pl
		matrix[pl.Col][pl.Row].Player = p[id]
	}

	return &Model{
		Matrix:  matrix,
		Players: p,
	}
}
