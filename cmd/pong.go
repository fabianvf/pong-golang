package main

import (
	"fmt"

	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"github.com/fabianvf/pong/pkg/future"
	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/text"
	"golang.org/x/image/font"

	"github.com/SolarLune/resolv/resolv"
)

var (
	paddleImage     *ebiten.Image
	ballImage       *ebiten.Image
	backgroundImage *ebiten.Image
	arcadeFont      font.Face
	smallArcadeFont font.Face
)

const (
	leftPaddle    = "left"
	rightPaddle   = "right"
	gameModeWait  = 0
	gameModePlay  = 1
	fontSize      = 32
	smallFontSize = fontSize / 2
	// backgroundURL = "https://cutewallpaper.org/21/black-cool-background/77+-Cool-Black-Background-Designs-on-WallpaperSafari.jpg"
)

func init() {
	var err error
	paddleImage, err = ebiten.NewImage(1, 1, ebiten.FilterDefault)
	if err != nil {
		log.Fatal(err)
	}
	ballImage, err = ebiten.NewImage(1, 1, ebiten.FilterDefault)
	if err != nil {
		log.Fatal(err)
	}
	// backgroundImage, err = ebitenutil.NewImageFromURL(backgroundURL)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	tt, err := truetype.Parse(fonts.ArcadeN_ttf)
	if err != nil {
		log.Fatal(err)
	}
	const dpi = 72
	arcadeFont = truetype.NewFace(tt, &truetype.Options{
		Size:    fontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	smallArcadeFont = truetype.NewFace(tt, &truetype.Options{
		Size:    smallFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
}

type Pair struct {
	X float64
	Y float64
}

func NewGame() *Game {
	// all values assume a 16 x 9 window
	g := &Game{
		GameMode:     gameModeWait,
		Score:        Pair{X: 0, Y: 0},
		LeftPaddle:   *resolv.NewRectangle(1, 8, 2, 4),
		RightPaddle:  *resolv.NewRectangle(15, 8, 2, 4),
		Ball:         *resolv.NewCircle(8, 4, 3),
		BallVelocity: Pair{X: 1, Y: 1},
	}
	g.Reset()
	return g
}

type Game struct {
	GameMode              int
	Score                 Pair
	LeftPaddle            resolv.Rectangle
	RightPaddle           resolv.Rectangle
	Ball                  resolv.Circle
	BallVelocity          Pair
	WindowHeight          int32
	WindowWidth           int32
	BallVelocityIncrement float64
	// MaxBallVelocity       float64
}

func (g *Game) Reset() {
	g.Ball.X = int32(g.WindowWidth / 2)
	g.Ball.Y = int32(g.WindowHeight / 2)
	g.Ball.Radius = int32(1 * float64(g.WindowWidth) / 60)
	g.BallVelocity.X = float64(g.WindowWidth) / 100
	g.BallVelocity.Y = float64(g.WindowHeight) / 100

	g.BallVelocityIncrement = g.BallVelocity.X * 1.1
	// g.MaxBallVelocity = g.BallVelocity.X * 2

	g.LeftPaddle.X = int32(g.WindowWidth / 16)
	g.LeftPaddle.Y = int32(g.WindowHeight / 2)
	g.LeftPaddle.W = 10
	g.LeftPaddle.H = int32(float64(g.WindowHeight) / 5)

	g.RightPaddle.X = int32(g.WindowWidth - g.WindowWidth/16)
	g.RightPaddle.Y = int32(g.WindowHeight / 2)
	g.RightPaddle.W = 10
	g.RightPaddle.H = int32(float64(g.WindowHeight) / 5)

}

func Abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func (g *Game) LeftPaddleUp() bool {
	if g.LeftPaddle.Y < 0 {
		return false
	}

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		return true
	}

	for _, touchID := range ebiten.TouchIDs() {
		x, y := ebiten.TouchPosition(touchID)
		if x < int(g.WindowWidth/2) {
			if y < int(g.LeftPaddle.Y+g.LeftPaddle.H/2) {
				return true
			}
		}
	}
	return false
}

func (g *Game) LeftPaddleDown() bool {
	if g.LeftPaddle.Y+g.LeftPaddle.H > g.WindowHeight {
		return false
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		return true
	}

	for _, touchID := range ebiten.TouchIDs() {
		x, y := ebiten.TouchPosition(touchID)
		if x < int(g.WindowWidth/2) {
			if y > int(g.LeftPaddle.Y+g.LeftPaddle.H/2) {
				return true
			}
		}
	}
	return false
}

func (g *Game) Update(screen *ebiten.Image) error {
	if g.GameMode == gameModeWait {
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			g.startGame()
		}
		return nil

	}
	if g.LeftPaddleUp() {
		g.LeftPaddle.Y += -int32(Abs(g.BallVelocity.Y))
	}
	if g.LeftPaddleDown() {
		g.LeftPaddle.Y += int32(Abs(g.BallVelocity.Y))
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) && g.RightPaddle.Y > 0 {
		g.RightPaddle.Y += -g.WindowHeight / 60
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) && g.RightPaddle.Y+g.RightPaddle.H < g.WindowHeight {
		g.RightPaddle.Y += g.WindowHeight / 60
	}
	if g.Ball.X+g.Ball.Radius > g.WindowWidth && g.BallVelocity.X > 0 {
		g.Score.Y += 1
		g.GameMode = gameModeWait
	}
	if g.Ball.X < 0 && g.BallVelocity.X < 0 {
		g.Score.X += 1
		g.GameMode = gameModeWait
	}
	if g.BallHitPaddle() {
		g.BallVelocity.X = -g.BallVelocity.X * 1.01
		g.BallVelocity.Y *= 1.01
	}

	if g.Ball.Y < 0 && g.BallVelocity.Y < 0 {
		g.BallVelocity.Y = -g.BallVelocity.Y
	}
	if g.Ball.Y+g.Ball.Radius > g.WindowHeight && g.BallVelocity.Y > 0 {
		g.BallVelocity.Y = -g.BallVelocity.Y
	}
	g.Ball.X += int32(g.BallVelocity.X)
	g.Ball.Y += int32(g.BallVelocity.Y)
	return nil
}

func (g *Game) PaddleDimensions() (int, int) {
	return int(g.WindowWidth / 100), int(g.WindowHeight / 5)
}

func (g *Game) BallHitPaddle() bool {

	oldW := g.LeftPaddle.W
	g.LeftPaddle.W = 0
	resolution := resolv.Resolve(&g.Ball, &g.LeftPaddle, int32(g.BallVelocity.X), int32(g.BallVelocity.Y))
	g.LeftPaddle.W = oldW
	if resolution.Colliding() {
		// fmt.Printf("%+v\n", resolution)
		return true
	}
	resolution = resolv.Resolve(&g.Ball, &g.RightPaddle, int32(g.BallVelocity.X), int32(g.BallVelocity.Y))
	if resolution.Colliding() {
		// fmt.Printf("%+v\n", resolution)
		return true
	}
	return false
}

func (g *Game) startGame() {
	g.Reset()
	g.GameMode = gameModePlay
}

func (g *Game) centerText(content string, face font.Face) (int, int) {
	centerX, centerY := int(g.WindowWidth/2), int(g.WindowHeight/2)

	stringSize := future.MeasureString(content, face)
	centerX -= stringSize.X / 2
	centerY -= stringSize.Y / 2

	return centerX, centerY

}

func (g *Game) Draw(screen *ebiten.Image) {
	// g.drawBackground(screen)
	screen.Fill(color.RGBA{0x80, 0xa0, 0xc0, 0xff})
	ebitenutil.DebugPrint(screen, fmt.Sprintf("%+v", g))
	switch g.GameMode {
	case gameModeWait:
		score := fmt.Sprintf("%+v - %+v", int(g.Score.X), int(g.Score.Y))
		x, y := g.centerText(score, arcadeFont)
		text.Draw(screen, score, arcadeFont, x, y, color.White)
		startMessage := "Press Space to Start"
		x, y = g.centerText(startMessage, smallArcadeFont)
		text.Draw(screen, startMessage, smallArcadeFont, x, y+20, color.White)
	case gameModePlay:
		g.drawPaddles(screen)
		g.drawBall(screen)
	}
}

// func (g *Game) drawBackground(screen *ebiten.Image) {
// 	backgroundOpts := ebiten.DrawImageOptions{}
// 	// backgroundOpts.GeoM.Scale(float64(g.WindowWidth), float64(g.WindowHeight))
// 	screen.DrawImage(backgroundImage, &backgroundOpts)
// }

func (g *Game) drawBall(screen *ebiten.Image) {
	ballImage.Fill(color.White)
	ballOpts := ebiten.DrawImageOptions{}
	ballOpts.GeoM.Scale(float64(g.Ball.Radius), float64(g.Ball.Radius))
	ballOpts.GeoM.Translate(float64(g.Ball.X), float64(g.Ball.Y))
	screen.DrawImage(ballImage, &ballOpts)
}

func (g *Game) drawPaddles(screen *ebiten.Image) {
	paddleImage.Fill(color.White)

	leftPaddleOpts := ebiten.DrawImageOptions{}
	leftPaddleOpts.GeoM.Scale(float64(g.LeftPaddle.W), float64(g.LeftPaddle.H))
	leftPaddleOpts.GeoM.Translate(float64(g.LeftPaddle.X), float64(g.LeftPaddle.Y))

	rightPaddleOpts := ebiten.DrawImageOptions{}
	rightPaddleOpts.GeoM.Scale(float64(g.RightPaddle.W), float64(g.RightPaddle.H))
	rightPaddleOpts.GeoM.Translate(float64(g.RightPaddle.X), float64(g.RightPaddle.Y))

	screen.DrawImage(paddleImage, &leftPaddleOpts)
	screen.DrawImage(paddleImage, &rightPaddleOpts)

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	if g.WindowWidth != int32(outsideWidth) || g.WindowHeight != int32(outsideHeight) {
		g.WindowWidth = int32(outsideWidth)
		g.WindowHeight = int32(outsideHeight)
		g.GameMode = gameModeWait
	}
	return outsideWidth, outsideHeight
}

func main() {
	fmt.Println("HELLO")
	// ebiten.SetWindowSize(1080, 720)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Pong, but shitty")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
