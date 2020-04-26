package main

import (
	"bytes"
	"fmt"
	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/inpututil"
	"github.com/hajimehoshi/ebiten/text"
	"github.com/tanema/gween"
	"golang.org/x/image/font"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	size = 50
)

func HexToF32(u uint32, id int) GameColor {
	b := float64(0xff&u) / 255
	g := float64(0xff&(u>>8)) / 255
	r := float64(0xff&(u>>16)) / 255
	return GameColor{r, g, b, id}
}

type GameColor struct {
	r  float64
	g  float64
	b  float64
	id int
}

var cols = 10
var rows = 10
var screenWidth = cols * size
var screenHeight = rows * size

var COLOR_NONE = HexToF32(0x000000, 0)
var COLOR_STONE = HexToF32(0x444444, 7)

var COLORS = []GameColor{
	HexToF32(0xfa3636, 1),
	HexToF32(0xedbc1e, 2),
	HexToF32(0x0abd38, 3),
	HexToF32(0x34fbf6, 4),
	HexToF32(0x321ecc, 5),
	HexToF32(0xcb18dd, 6),
}

// Tile represents an image.
type Tile struct {
	image          *ebiten.Image
	explodable     bool
	x              int
	y              int
	scaleX, scaleY float64
	color          GameColor
	alpha          float64
	width, height  int
	Col, Row       int
}

// Draw draws the sprite.
func (s *Tile) Draw(screen *ebiten.Image, dx, dy int, alpha float64, offset float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(s.scaleX*.92, s.scaleY*.92) //s.width, s.height)
	op.GeoM.Translate(float64(s.x+dx), float64(s.y+dy)-offset)
	op.GeoM.Rotate(0)
	op.ColorM.Scale(s.color.r, s.color.g, s.color.b, alpha)
	screen.DrawImage(s.image, op)
}

// StrokeSource represents a input device to provide strokes.
type StrokeSource interface {
	Position() (int, int)
	IsJustReleased() bool
}

// MouseStrokeSource is a StrokeSource implementation of mouse.
type MouseStrokeSource struct{}

func (m *MouseStrokeSource) Position() (int, int) {
	return ebiten.CursorPosition()
}

func (m *MouseStrokeSource) IsJustReleased() bool {
	return inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
}

// TouchStrokeSource is a StrokeSource implementation of touch.
type TouchStrokeSource struct {
	ID int
}

func (t *TouchStrokeSource) Position() (int, int) {
	return ebiten.TouchPosition(t.ID)
}

func (t *TouchStrokeSource) IsJustReleased() bool {
	return inpututil.IsTouchJustReleased(t.ID)
}

// Stroke manages the current drag state by mouse.
type Stroke struct {
	source StrokeSource

	// initX and initY represents the position when dragging starts.
	initX int
	initY int

	// currentX and currentY represents the current position
	currentX int
	currentY int

	released bool
}

func NewStroke(source StrokeSource) *Stroke {
	cx, cy := source.Position()
	return &Stroke{
		source:   source,
		initX:    cx,
		initY:    cy,
		currentX: cx,
		currentY: cy,
	}
}

func (s *Stroke) Update() {
	if s.released {
		return
	}
	if s.source.IsJustReleased() {
		s.released = true
		return
	}
	x, y := s.source.Position()
	s.currentX = x
	s.currentY = y
}

func (s *Stroke) IsReleased() bool {
	return s.released
}

func (s *Stroke) Position() (int, int) {
	return s.currentX, s.currentY
}

func (s *Stroke) PositionDiff() (int, int) {
	dx := s.currentX - s.initX
	dy := s.currentY - s.initY
	return dx, dy
}

type GameState int

const (
	IDLE GameState = iota + 1
	COOLDOWN
	ACTING
	GAME_OVER
)

func (s GameState) Name() string {
	switch s {
	case IDLE:
		return "IDLE"
	case COOLDOWN:
		return "COOLDOWN"
	case ACTING:
		return "ACTING"
	case GAME_OVER:
		return "GAME_OVER"
	default:
		return fmt.Sprintf("N/A(%d)", s)
	}
}

type Game struct {
	State   GameState
	Model   *Model
	Line    *Nine
	strokes map[*Stroke]struct{}

	sprites             map[*Tile]struct{}
	Matrix              [][]Cell
	Cols, Rows          int
	Tweens              map[*gween.Tween]Action
	ThrowerIndex        int
	ThrowColumn         [][]int
	SlidedColumns       int
	LastSwipeWaitsThrow bool
	FallsInRound        int
	NoStones            int
	score               int
	ScoreLabel          *ebiten.Image
	level               int
	LevelLabel          *ebiten.Image
	lastStepUp          int
	yOffset             float64
}

var theGame *Game

var bgImage, eImmF *ebiten.Image
var Font font.Face

func init() {

	//Load()

	dat, err := ebitenutil.OpenFile("Teko-Light.ttf")
	//dat, err := ioutil.ReadFile("Teko-Light.ttf")

	buf := new(bytes.Buffer)
	buf.ReadFrom(dat)

	tt, err := truetype.Parse(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	Font = truetype.NewFace(tt, &truetype.Options{
		Size:       50,
		DPI:        dpi,
		SubPixelsX: 100,
		Hinting:    font.HintingFull,
	})

	dot, _, err := ebitenutil.NewImageFromFile("circle.png", ebiten.FilterDefault)
	//imgF, _, err := image.Decode(bufio.NewReader(fileF))
	if err != nil {
		log.Fatal(err)
	}

	nine := Nine{
		images: dot,
		alpha:  1,
		R:      1, G: 1, B: 1, Scale: .15,
		positions: [4][2]int{{0, 0}, {56, 56}, {57, 57}, {112, 112}}}

	//fileF, err := os.Open("frame.png")
	eImmF, _, err = ebitenutil.NewImageFromFile("tile.png", ebiten.FilterDefault)
	//imgF, _, err := image.Decode(bufio.NewReader(fileF))
	if err != nil {
		log.Fatal(err)
	}
	model, _ := Load()

	theGame = &Game{
		strokes:    map[*Stroke]struct{}{},
		Tweens:     make(map[*gween.Tween]Action),
		Model:      model,
		Line:       &nine,
		Cols:       len(model.Matrix[0]),
		Rows:       len(model.Matrix),
		State:      IDLE,
		ScoreLabel: prepareTextImage("00000"),
		LevelLabel: prepareTextImage("1"),
		score:      0,
	}

}

func (g *Game) updateStroke(stroke *Stroke) {
	stroke.Update()
	xDif, yDif := stroke.PositionDiff()
	if math.Abs(float64(xDif)) > size/2 {
		stroke.released = true
	}

	if math.Abs(float64(yDif)) > size/2 {
		stroke.released = true
		// send event
	}

	if !stroke.IsReleased() {
		return
	}

	//-----------------------------------------------------

	//newX, _ := stroke.PositionDiff()
	//dx := float32(newX - 0)

}

func prepareTextImage(s string) *ebiten.Image {
	image, _ := ebiten.NewImage(300, 150, ebiten.FilterLinear)
	//image.Fill(color.RGBA{255, 0, 0, 255})
	text.Draw(image, s, Font, 5, 80, color.White)
	return image
}

func (g *Game) update(screen *ebiten.Image) error {
	//log.Print("dddddd")

	// tween
	for t, a := range g.Tweens {
		curr, finished := t.Update(0.02)
		if a.onChange != nil {
			a.onChange(curr)
		}
		if finished {
			for _, onFinish := range a.onFinish {
				onFinish()
			}
			for _, next := range a.nexts {
				next(g)
			}
			delete(g.Tweens, t)
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		s := NewStroke(&MouseStrokeSource{})
		//s.SetDraggingObject(g.spriteAt(s.Position()))
		g.strokes[s] = struct{}{}
	}

	for _, id := range inpututil.JustPressedTouchIDs() {
		s := NewStroke(&TouchStrokeSource{id})
		//s.SetDraggingObject(g.spriteAt(s.Position()))
		g.strokes[s] = struct{}{}
	}

	for s := range g.strokes {
		g.updateStroke(s)
		if s.IsReleased() {
			delete(g.strokes, s)
		}
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	e := screen.Fill(color.RGBA{70, 70, 70, 255})
	if e != nil {
		log.Printf("%v", e)
	}

	for c := 0; c < len(g.Matrix); c++ {
		for r := 0; r < len(g.Matrix[0]); r++ {
			/*
				if g.Matrix[c][r].Type == O {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(.28, .28) //s.width, s.height)
					op.GeoM.Translate(float64(c*size)-5, float64((rows-r-1)*size)-5-g.yOffset)
					op.GeoM.Rotate(0)
					op.ColorM.Scale(0.05, .05, .05, 1)
					screen.DrawImage(bgImage, op)
				}
				if g.Matrix[c][r].Type == C {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(.18, .18) //s.width, s.height)
					op.GeoM.Translate(float64(c*size)+5, float64((rows-r-1)*size)+5-g.yOffset)
					op.GeoM.Rotate(0)
					op.ColorM.Scale(0.5, .5, .05, 1)
					screen.DrawImage(bgImage, op)
				}

			*/
		}
	}

	theGame.Line.SetPosition(100,100)
	theGame.Line.SetSize(17,300)
	theGame.Line.Draw(screen)

	//g.Score
	op := &ebiten.DrawImageOptions{}
	//op.GeoM.Scale(.35, .35) //s.width, s.height)
	op.GeoM.Translate(251, 10)
	op.GeoM.Rotate(0)
	//op.ColorM.Scale(0.05, .05, .05, 1)
	screen.DrawImage(g.ScoreLabel, op)
	op = &ebiten.DrawImageOptions{}
	//op.GeoM.Scale(.35, .35) //s.width, s.height)
	op.GeoM.Scale(.7, 0.7)
	op.GeoM.Translate(252, 60)
	op.GeoM.Rotate(0)
	screen.DrawImage(g.LevelLabel, op)

	ebitenutil.DebugPrintAt(screen, g.State.Name(), 300, 0)

	return nil
}

func main() {
	if err := ebiten.Run(theGame.update, screenWidth, screenHeight, 1, "Left-Right-Swipe"); err != nil {
		log.Fatal(err)
	}
}
