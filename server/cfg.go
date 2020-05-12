package server

import (
	"bufio"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/zucenko/roader/model"
	"io"
	"log"
)

func Load() (m *model.Model, e error) {
	file, fileErr := ebitenutil.OpenFile("data/data_1.txt")
	if fileErr != nil {
		e = fileErr
		return
	}
	defer file.Close()
	if e != nil {
		log.Printf("failed opening file: %s", e)
		return
	}
	cells, players, err := read(file)
	if err != nil {
		return nil, err
	}
	transformed := swap(&cells)

	keys := make([]int32, 0, len(players))
	for k := range players {
		keys = append(keys, k)
	}

	return &model.Model{Matrix: *transformed, Players: players, PlayerKeys: keys}, nil
}

func swap(oldCells *[][]*model.Cell) *[][]*model.Cell {
	newCells := make([][]*model.Cell, 0)
	for c := 0; c < len((*oldCells)[0]); c++ {
		col := make([]*model.Cell, 0)
		for r := 0; r < len(*oldCells); r++ {
			cell := (*oldCells)[r][c]
			col = append(col, cell)
		}
		newCells = append(newCells, col)
	}
	return &newCells
}

func read(reader io.Reader) (cels [][]*model.Cell, players map[int32]*model.Player, err error) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	ReadLines := make([][]*model.Cell, 0)
	players = make(map[int32]*model.Player)
	portals := make(map[int32][2]int)
	rows := 0
	matrixRow := 0
	matrixCol := 0

	for scanner.Scan() {
		s := scanner.Text()
		matrixCol = 0
		if rows%2 == 0 {
			// real line
			Line := make([]*model.Cell, 0)
			for i, char := range s {
				if i%2 == 0 {
					//log.Printf("reading r:%d c:%d", rows/2, i/2)
					// real cell
					cell := &model.Cell{Col: matrixCol, Row: matrixRow}
					Line = append(Line, cell)
					switch char {
					case '*':
						cell.Diamond = true
					case 'K':
						cell.Key = true
					case 'A', 'B', 'C':
						p := model.Player{Id: char, Col: cell.Col, Row: cell.Row}
						players[char] = &p
						cell.Player = &p
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						prev, found := portals[char]
						if found {
							var pevPortalCell *model.Cell
							// is it by accident yet this line not ever added?
							if prev[1] == len(ReadLines) {
								pevPortalCell = Line[prev[0]]
							} else {
								pevPortalCell = ReadLines[prev[1]][prev[0]]
							}
							pevPortalCell.Portal = &model.Portal{Target: cell}
							cell.Portal = &model.Portal{Target: pevPortalCell}
							delete(portals, char)
						} else {
							portals[char] = [2]int{matrixCol, matrixRow}
						}
					}

					if matrixCol == 0 {
						cell.Paths[2] = &model.Path{Dir: 2, Target: nil, Wall: true}
					} else {
						// connect prev left
						prevCell := Line[matrixCol-1]
						prevCell.Paths[0].Target = cell
						cell.Paths[2] = &model.Path{
							Dir:    2,
							Target: prevCell,
							Wall:   prevCell.Paths[0].Wall,
							Lock:   prevCell.Paths[0].Lock}
					}
					if matrixRow == 0 {
						cell.Paths[3] = &model.Path{Dir: 3, Target: nil, Wall: true}
					} else {
						// connect prev up
						prevCell := ReadLines[matrixRow-1][matrixCol]
						prevCell.Paths[1].Target = cell
						cell.Paths[3] = &model.Path{
							Dir:    3,
							Target: prevCell,
							Wall:   prevCell.Paths[1].Wall,
							Lock:   prevCell.Paths[1].Lock}
					}
					cell.Paths[1] = &model.Path{Dir: 1, Wall: true}
				} else {
					switch char {
					case '|':
						Line[matrixCol].Paths[0] = &model.Path{Dir: 0, Wall: true}
					case ':':
						Line[matrixCol].Paths[0] = &model.Path{Dir: 0, Wall: true, Lock: true}
					case ' ':
						Line[matrixCol].Paths[0] = &model.Path{Dir: 0}
					case '.':
						//end line
						Line[matrixCol].Paths[0] = &model.Path{Dir: 0, Wall: true}
					}
					matrixCol++
				}

			}
			ReadLines = append(ReadLines, Line)
		} else {
			// bottom wall
			for i, char := range s {
				if i%2 == 0 {
					cell := ReadLines[matrixRow][matrixCol]
					switch char {
					case '-':
						cell.Paths[1].Wall = true
					case '~':
						cell.Paths[1].Wall = true
						cell.Paths[1].Lock = true
					case ' ':
						cell.Paths[1].Wall = false
					}
					matrixCol++
				}
			}
			matrixRow++
		}
		rows++
	}

	return ReadLines, players, nil
}
