package main

import (
	"bufio"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"io"
	"log"
)

func Load() (m *Model, e error) {
	file, fileErr := ebitenutil.OpenFile("data.txt")
	if fileErr != nil {
		e = fileErr
		return
	}
	defer file.Close()
	if e != nil {
		log.Printf("failed opening file: %s", e)
		return
	}
	return read(file)
}

func read(reader io.Reader) (m *Model, err error) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	ReadLines := make([][]Cell, 0)
	players := make(map[int32]Player)
	portals := make(map[int32][2]int)
	rows := 0
	matrixRow := 0
	matrixCol := 0

	for scanner.Scan() {
		s := scanner.Text()
		matrixCol = 0
		if rows%2 == 0 {
			// real line
			Line := make([]Cell, 0)
			for i, char := range s {
				if i%2 == 0 {
					// real cell
					Line = append(Line, Cell{col: matrixCol, row: matrixRow})
					cell := &Line[len(Line)-1]
					switch char {
					case '*':
						cell.Diamond = true
					case 'A', 'B', 'C':
						players[char] = Player{Color: COLORS[char-'A'], col: cell.col, row: cell.row}
					case '0', '1', '2', '3', '4':
						prev, found := portals[char]
						if found {
							pevPortalCell := &ReadLines[prev[1]][prev[0]]
							pevPortalCell.Portal = Portal{Target: cell}
							cell.Portal = Portal{Target: pevPortalCell}
							delete(portals, char)
						} else {
							portals[char] = [2]int{matrixCol, matrixRow}
						}
					}
					if matrixCol > 0 {
						// connect prev left
						prevCell := &Line[matrixCol-1]
						prevCell.Paths[0].Target = cell
						cell.Paths[2] = &Path{Target: prevCell, Wall: prevCell.Paths[0].Wall}
					}
					if matrixRow > 0 {
						// connect prev up
						prevCell := &ReadLines[matrixRow-1][matrixCol]
						prevCell.Paths[1].Target = cell
						cell.Paths[3] = &Path{Target: prevCell, Wall: prevCell.Paths[1].Wall}
					}
				} else {
					switch char {
					case '|':
						Line[matrixCol].Paths[0] = &Path{Wall: true}
					case ' ':
						Line[matrixCol].Paths[0] = &Path{}
					case '.':
						//end line
					}
					matrixCol++
				}

			}
			ReadLines = append(ReadLines, Line)
		} else {
			// bottom wall
			for i, char := range s {
				if i%2 == 0 {
					cell := &ReadLines[matrixRow][matrixCol]
					switch char {
					case '-':
						cell.Paths[1] = &Path{Wall: true}
					case ' ':
						cell.Paths[1] = &Path{}
					}
					matrixCol++
				}
			}
			matrixRow++
		}
		rows++
	}
	return &Model{Matrix: ReadLines},nil
}
