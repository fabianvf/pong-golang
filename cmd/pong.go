package main

import (
	"fmt"
	"math"

	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"github.com/fabianvf/pong/pkg/future"
	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/inpututil"
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
	gameModePause = 2
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
	MaxBallVelocity       float64
	MinBallVelocity       float64
	PaddleSpeed           int32
	BallSpeed             float64
}

func (g *Game) Reset() {
	g.Ball.X = int32(g.WindowWidth / 2)
	g.Ball.Y = int32(g.WindowHeight / 2)
	g.Ball.Radius = int32(float64(g.WindowWidth) / 60)
	g.BallSpeed = float64(g.WindowWidth) / 100

	g.BallVelocity.X = g.BallSpeed
	g.BallVelocity.Y = g.BallSpeed

	g.BallVelocityIncrement = g.BallSpeed / 10
	g.MaxBallVelocity = g.BallSpeed * 3
	g.MinBallVelocity = g.BallSpeed

	g.LeftPaddle.X = int32(g.WindowWidth / 16)
	g.LeftPaddle.Y = int32(g.WindowHeight / 2)
	g.LeftPaddle.W = int32(float64(g.WindowWidth) / 60)
	g.LeftPaddle.H = int32(float64(g.WindowHeight) / 5)

	g.RightPaddle.X = int32(g.WindowWidth - g.WindowWidth/16)
	g.RightPaddle.Y = int32(g.WindowHeight / 2)
	g.RightPaddle.W = int32(float64(g.WindowWidth) / 60)
	g.RightPaddle.H = int32(float64(g.WindowHeight) / 5)
	g.PaddleSpeed = int32(float64(g.WindowHeight) / 60)
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

func (g *Game) RightPaddleUp() bool {
	if g.RightPaddle.Y < 0 {
		return false
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		return true
	}

	for _, touchID := range ebiten.TouchIDs() {
		x, y := ebiten.TouchPosition(touchID)
		if x > int(g.WindowWidth/2) {
			if y < int(g.RightPaddle.Y+g.RightPaddle.H/2) {
				return true
			}
		}
	}
	return false
}

func (g *Game) RightPaddleDown() bool {
	if g.RightPaddle.Y+g.RightPaddle.H > g.WindowHeight {
		return false
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		return true
	}

	for _, touchID := range ebiten.TouchIDs() {
		x, y := ebiten.TouchPosition(touchID)
		if x > int(g.WindowWidth/2) {
			if y > int(g.RightPaddle.Y+g.RightPaddle.H/2) {
				return true
			}
		}
	}
	return false
}

func keyPressStartGame() bool {
	keys := []ebiten.Key{ebiten.KeySpace, ebiten.KeyW, ebiten.KeyS, ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyEnter}
	for _, key := range keys {
		if inpututil.IsKeyJustPressed(key) {
			return true
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	if len(inpututil.JustPressedTouchIDs()) > 0 {
		return true
	}
	return false
}

func (g *Game) Update(screen *ebiten.Image) error {
	if g.GameMode == gameModePause {
		if keyPressStartGame() {
			g.GameMode = gameModePlay
		}
		return nil
	}
	if g.GameMode == gameModeWait {
		if keyPressStartGame() {
			g.startGame()
		}
		return nil

	}
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		g.GameMode = gameModePause
		return nil
	}

	if g.LeftPaddleUp() {
		g.LeftPaddle.Y += -g.PaddleSpeed
	}
	if g.LeftPaddleDown() {
		g.LeftPaddle.Y += g.PaddleSpeed
	}
	if g.RightPaddleUp() {
		g.RightPaddle.Y += -g.PaddleSpeed
	}
	if g.RightPaddleDown() {
		g.RightPaddle.Y += g.PaddleSpeed
	}

	if g.Ball.X+g.Ball.Radius > g.WindowWidth && g.BallVelocity.X > 0 {
		g.Score.Y += 1
		g.GameMode = gameModeWait
	}
	if g.Ball.X < 0 && g.BallVelocity.X < 0 {
		g.Score.X += 1
		g.GameMode = gameModeWait
	}
	g.HandleBallPaddleCollision()

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

func GetBounceVelocity(Paddle resolv.Rectangle, Ball resolv.Circle, maxSpeed float64) (float64, float64) {
	relativeIntersect := float64(Paddle.Y) + (float64(Paddle.H) / 2) - (float64(Ball.Y) + float64(Ball.Radius)/2)
	normalizedRelativeIntersect := (relativeIntersect / (float64(Paddle.H) / 2))
	angle := normalizedRelativeIntersect * math.Pi / 4
	return Abs(math.Cos(angle)) * maxSpeed * Abs(angle), -math.Sin(angle) * maxSpeed * Abs(angle)
}

func (g *Game) HandleBallPaddleCollision() {
	oldX := g.LeftPaddle.X
	g.LeftPaddle.X -= g.LeftPaddle.W
	resolution := resolv.Resolve(&g.Ball, &g.LeftPaddle, int32(g.BallVelocity.X), int32(g.BallVelocity.Y))
	g.LeftPaddle.X = oldX

	if resolution.Colliding() && g.BallVelocity.X < 0 {
		g.BallSpeed += g.BallVelocityIncrement
		vx, vy := GetBounceVelocity(g.LeftPaddle, g.Ball, g.MaxBallVelocity)
		g.BallVelocity.X = vx
		g.BallVelocity.Y = vy
	}

	oldX = g.RightPaddle.X
	g.RightPaddle.X += g.RightPaddle.W
	resolution = resolv.Resolve(&g.Ball, &g.RightPaddle, int32(g.BallVelocity.X), int32(g.BallVelocity.Y))
	g.RightPaddle.X = oldX

	if resolution.Colliding() && g.BallVelocity.X > 0 {
		g.BallSpeed += g.BallVelocityIncrement
		vx, vy := GetBounceVelocity(g.RightPaddle, g.Ball, g.MaxBallVelocity)
		g.BallVelocity.X = -(vx)
		g.BallVelocity.Y = vy
	}

	if Abs(g.BallVelocity.X) < g.MinBallVelocity {
		multiplier := 1.0
		if g.BallVelocity.X < 0 {
			multiplier = -1
		}
		g.BallVelocity.X = g.MinBallVelocity * multiplier
	}
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

func (g *Game) drawStart(screen *ebiten.Image) {
	startMessage := "Press to Start"
	x, y := g.centerText(startMessage, smallArcadeFont)
	text.Draw(screen, startMessage, smallArcadeFont, x, y+20, color.Black)
}

func (g *Game) Draw(screen *ebiten.Image) {
	// g.drawBackground(screen)
	screen.Fill(color.RGBA{0x80, 0xa0, 0xc0, 0xff})
	// ebitenutil.DebugPrint(screen, fmt.Sprintf("%+v", g))
	score := fmt.Sprintf("%+v - %+v", int(g.Score.X), int(g.Score.Y))
	x, y := g.centerText(score, arcadeFont)
	text.Draw(screen, score, arcadeFont, x, y, color.Black)
	switch g.GameMode {
	case gameModeWait:
		g.drawStart(screen)
	case gameModePlay:
		g.drawPaddles(screen)
		g.drawBall(screen)
	case gameModePause:
		g.drawStart(screen)
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
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Pong, but shitty")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
