package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"math"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/mp3"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/inpututil"
	"github.com/hajimehoshi/ebiten/text"

	"github.com/SolarLune/resolv/resolv"

	"github.com/fabianvf/pong-golang/pkg/future"
	raudio "github.com/fabianvf/pong-golang/pkg/resources/audio"
	rimage "github.com/fabianvf/pong-golang/pkg/resources/images"
)

var (
	paddleImage      *ebiten.Image
	ballImage        *ebiten.Image
	backgroundImage  *ebiten.Image
	arcadeFont       font.Face
	smallArcadeFont  font.Face
	audioContext     *audio.Context
	backgroundPlayer *audio.Player
	hitPlayer        *audio.Player
	cpuprofile       = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile       = flag.String("memprofile", "", "write memory profile to file")
)

const (
	leftPaddle    = "left"
	rightPaddle   = "right"
	gameModeWait  = 0
	gameModePlay  = 1
	gameModePause = 2
	fontSize      = 32
	smallFontSize = fontSize / 2
	trailPolygons = 1000
)

func init() {
	var err error
	img, _, err := image.Decode(bytes.NewReader(rimage.Background_png))
	if err != nil {
		log.Fatal(err)
	}
	backgroundImage, err = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
	if err != nil {
		log.Fatal(err)
	}

	paddleImage, err = ebiten.NewImage(1, 1, ebiten.FilterDefault)
	if err != nil {
		log.Fatal(err)
	}
	ballImage, err = ebiten.NewImage(1, 1, ebiten.FilterDefault)
	if err != nil {
		log.Fatal(err)
	}

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

	audioContext, _ = audio.NewContext(22050 / 2)

	backgroundAudio, err := mp3.Decode(audioContext, audio.BytesReadSeekCloser(raudio.Background_mp3))
	if err != nil {
		log.Fatal(err)
	}
	backgroundPlayer, err = audio.NewPlayer(audioContext, backgroundAudio)
	if err != nil {
		log.Fatal(err)
	}
	hitAudio, err := mp3.Decode(audioContext, audio.BytesReadSeekCloser(raudio.Hit_mp3))
	if err != nil {
		log.Fatal(err)
	}
	hitPlayer, err = audio.NewPlayer(audioContext, hitAudio)
	if err != nil {
		log.Fatal(err)
	}
}

type Pair struct {
	X float64
	Y float64
}

type TrailElement struct {
	Radius   float64
	Coord    Pair
	Velocity Pair
	Angle    float64
	Speed    float64
}

func (t *TrailElement) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	options.GeoM.Reset()
	options.GeoM.Scale(t.Radius, t.Radius+t.Speed)
	options.GeoM.Rotate(t.Angle)
	options.GeoM.Translate(t.Coord.X, t.Coord.Y)

	screen.DrawImage(ballImage, options)
}

func NewTrail() Trail {
	elements := make([]TrailElement, trailPolygons)
	return Trail{elements: elements}
}

type Trail struct {
	elements     []TrailElement
	currentAngle float64
}

func (t *Trail) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	for _, element := range t.elements {
		element.Draw(screen, options)
	}
}

func (t *Trail) UpdateAngle(velocity Pair) {
	t.currentAngle = math.Atan2(velocity.Y, velocity.X) - (3*math.Pi)/2
}

func (t *Trail) Add(b *Ball) {
	newElement := TrailElement{
		Radius:   float64(b.Radius) * 0.5,
		Coord:    b.Coord,
		Velocity: b.Velocity,
		Angle:    t.currentAngle,
		Speed:    math.Sqrt(math.Pow(b.Velocity.Y, 2) + math.Pow(b.Velocity.X, 2)),
	}
	newElement.Coord.Y += newElement.Radius / 2
	if newElement.Velocity.X < 0 {
		newElement.Coord.Y += newElement.Radius
	}
	t.elements = append(
		[]TrailElement{newElement},
		t.elements[:len(t.elements)-1]...)
	for i, _ := range t.elements {
		t.elements[i].Radius = t.elements[i].Radius * 0.9
	}
}

func NewBall(radius int32, x, y, vy, vx, minV, maxV, speed float64) *Ball {
	return &Ball{
		Radius:         radius,
		Coord:          Pair{X: x, Y: y},
		Velocity:       Pair{X: vx, Y: vy},
		VelocityBounds: Pair{X: minV, Y: maxV},
		BaseSpeed:      speed,
		boundingBox:    *resolv.NewCircle(int32(x), int32(y), int32(radius)),
		trail:          NewTrail(),
	}
}

type Ball struct {
	Radius         int32
	Coord          Pair
	Velocity       Pair
	VelocityBounds Pair
	BaseSpeed      float64
	boundingBox    resolv.Circle
	trail          Trail
}

func (b *Ball) Draw(screen *ebiten.Image) {
	ballImage.Fill(color.White)
	ballOpts := ebiten.DrawImageOptions{}
	ballOpts.GeoM.Scale(float64(b.Radius), float64(b.Radius))
	ballOpts.GeoM.Translate(float64(b.Coord.X), float64(b.Coord.Y))
	screen.DrawImage(ballImage, &ballOpts)
	b.trail.Draw(screen, &ballOpts)
}

func (b *Ball) BoundingBox() *resolv.Circle {
	b.boundingBox.X = int32(b.Coord.X)
	b.boundingBox.Y = int32(b.Coord.Y)
	b.boundingBox.Radius = b.Radius
	return &b.boundingBox
}

func NewGame() *Game {
	backgroundPlayer.Play()
	g := &Game{
		GameMode:    gameModeWait,
		Score:       Pair{X: 0, Y: 0},
		LeftPaddle:  *resolv.NewRectangle(1, 8, 2, 4),
		RightPaddle: *resolv.NewRectangle(15, 8, 2, 4),
		Ball:        *NewBall(3, 8, 4, 0, 0, 0, 0, 0),
	}
	g.Reset()
	return g
}

type Game struct {
	GameMode     int
	Score        Pair
	LeftPaddle   resolv.Rectangle
	RightPaddle  resolv.Rectangle
	Ball         Ball
	WindowHeight int32
	WindowWidth  int32
	PaddleSpeed  int32
}

func (g *Game) Reset() {
	g.Ball = *NewBall(3, 8, 4, 0, 0, 0, 0, 0)
	g.Ball.Coord.X = float64(g.WindowWidth) / 2
	g.Ball.Coord.Y = float64(g.WindowHeight) / 2
	g.Ball.Radius = int32(float64(g.WindowWidth) / 60)
	g.Ball.BaseSpeed = float64(g.WindowWidth) / 100

	g.Ball.Velocity.X = g.Ball.BaseSpeed
	g.Ball.Velocity.Y = g.Ball.BaseSpeed

	g.Ball.VelocityBounds.X = g.Ball.BaseSpeed
	g.Ball.VelocityBounds.Y = g.Ball.BaseSpeed * 3

	g.LeftPaddle.X = int32(g.WindowWidth / 16)
	g.LeftPaddle.Y = int32(g.WindowHeight / 2)
	g.LeftPaddle.W = int32(float64(g.WindowWidth) / 60)
	g.LeftPaddle.H = int32(float64(g.WindowHeight) / 5)

	g.RightPaddle.X = int32(g.WindowWidth - g.WindowWidth/16)
	g.RightPaddle.Y = int32(g.WindowHeight / 2)
	g.RightPaddle.W = int32(float64(g.WindowWidth) / 60)
	g.RightPaddle.H = int32(float64(g.WindowHeight) / 5)
	g.PaddleSpeed = int32(float64(g.WindowHeight) / 60)
	g.Ball.trail.UpdateAngle(g.Ball.Velocity)
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

	if int32(g.Ball.Coord.X)+g.Ball.Radius > g.WindowWidth && g.Ball.Velocity.X > 0 {
		g.Score.Y += 1
		// g.Ball.Velocity.X *= -1
		g.GameMode = gameModeWait
	}
	if g.Ball.Coord.X < 0 && g.Ball.Velocity.X < 0 {
		g.Score.X += 1
		// g.Ball.Velocity.X *= -1
		g.GameMode = gameModeWait
	}
	g.HandleBallPaddleCollision()

	if g.Ball.Coord.Y < 0 && g.Ball.Velocity.Y < 0 {
		g.Ball.Velocity.Y = -g.Ball.Velocity.Y
		g.Ball.trail.UpdateAngle(g.Ball.Velocity)
	}
	if int32(g.Ball.Coord.Y)+g.Ball.Radius > g.WindowHeight && g.Ball.Velocity.Y > 0 {
		g.Ball.Velocity.Y = -g.Ball.Velocity.Y
		g.Ball.trail.UpdateAngle(g.Ball.Velocity)
	}
	g.Ball.Coord.X += g.Ball.Velocity.X
	g.Ball.Coord.Y += g.Ball.Velocity.Y
	g.Ball.trail.Add(&g.Ball)
	return nil
}

func (g *Game) PaddleDimensions() (int, int) {
	return int(g.WindowWidth / 100), int(g.WindowHeight / 5)
}

func GetBounceVelocity(Paddle *resolv.Rectangle, Ball *resolv.Circle, maxSpeed float64) (float64, float64) {
	relativeIntersect := float64(Paddle.Y) + (float64(Paddle.H) / 2) - (float64(Ball.Y) + float64(Ball.Radius)/2)
	normalizedRelativeIntersect := (relativeIntersect / (float64(Paddle.H) / 2))
	angle := normalizedRelativeIntersect * math.Pi / 4
	return Abs(math.Cos(angle)) * maxSpeed * Abs(angle), -math.Sin(angle) * maxSpeed * Abs(angle)
}

func (g *Game) HandleBallPaddleCollision() {
	oldX := g.LeftPaddle.X
	g.LeftPaddle.X -= g.LeftPaddle.W
	resolution := resolv.Resolve(g.Ball.BoundingBox(), &g.LeftPaddle, int32(g.Ball.Velocity.X), int32(g.Ball.Velocity.Y))
	g.LeftPaddle.X = oldX

	if resolution.Colliding() && g.Ball.Velocity.X < 0 {
		vx, vy := GetBounceVelocity(&g.LeftPaddle, g.Ball.BoundingBox(), g.Ball.VelocityBounds.Y)
		g.Ball.Velocity.X = vx
		g.Ball.Velocity.Y = vy
		hitPlayer.Rewind()
		hitPlayer.Play()
		g.Ball.trail.UpdateAngle(g.Ball.Velocity)
	}

	oldX = g.RightPaddle.X
	g.RightPaddle.X += g.RightPaddle.W
	resolution = resolv.Resolve(g.Ball.BoundingBox(), &g.RightPaddle, int32(g.Ball.Velocity.X), int32(g.Ball.Velocity.Y))
	g.RightPaddle.X = oldX

	if resolution.Colliding() && g.Ball.Velocity.X > 0 {
		vx, vy := GetBounceVelocity(&g.RightPaddle, g.Ball.BoundingBox(), g.Ball.VelocityBounds.Y)
		g.Ball.Velocity.X = -(vx)
		g.Ball.Velocity.Y = vy
		hitPlayer.Rewind()
		hitPlayer.Play()
		g.Ball.trail.UpdateAngle(g.Ball.Velocity)
	}

	if Abs(g.Ball.Velocity.X) < g.Ball.VelocityBounds.X {
		multiplier := 1.0
		if g.Ball.Velocity.X < 0 {
			multiplier = -1
		}
		g.Ball.Velocity.X = g.Ball.VelocityBounds.X * multiplier
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
	g.drawBackground(screen)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %+v, TPS: %+v", ebiten.CurrentFPS(), ebiten.CurrentTPS()))
	score := fmt.Sprintf("%+v - %+v", int(g.Score.X), int(g.Score.Y))
	x, y := g.centerText(score, arcadeFont)
	text.Draw(screen, score, arcadeFont, x, y, color.Black)
	switch g.GameMode {
	case gameModeWait:
		g.drawStart(screen)
	case gameModePlay:
		g.drawPaddles(screen)
		g.Ball.Draw(screen)
	case gameModePause:
		g.drawStart(screen)
		g.drawPaddles(screen)
		g.Ball.Draw(screen)
	}
}

func (g *Game) drawBackground(screen *ebiten.Image) {
	backgroundOpts := ebiten.DrawImageOptions{}
	w, h := backgroundImage.Size()
	backgroundOpts.GeoM.Scale(float64(g.WindowWidth)/float64(w), float64(g.WindowHeight)/float64(h))
	screen.DrawImage(backgroundImage, &backgroundOpts)
}

func (g *Game) drawPaddles(screen *ebiten.Image) {
	paddleImage.Fill(color.White)

	paddleOpts := ebiten.DrawImageOptions{}

	paddleOpts.GeoM.Scale(float64(g.LeftPaddle.W), float64(g.LeftPaddle.H))
	paddleOpts.GeoM.Translate(float64(g.LeftPaddle.X), float64(g.LeftPaddle.Y))
	screen.DrawImage(paddleImage, &paddleOpts)

	paddleOpts.GeoM.Reset()
	paddleOpts.GeoM.Scale(float64(g.RightPaddle.W), float64(g.RightPaddle.H))
	paddleOpts.GeoM.Translate(float64(g.RightPaddle.X), float64(g.RightPaddle.Y))
	screen.DrawImage(paddleImage, &paddleOpts)

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
	var cf, mf *os.File
	var err error

	flag.Parse()
	if *cpuprofile != "" {
		cf, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(cf)
	}
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM) // subscribe to system signals
	onKill := func(c chan os.Signal) {
		select {
		case <-c:
			defer cf.Close()
			defer pprof.StopCPUProfile()
			defer os.Exit(0)
			if *memprofile != "" {
				mf, err = os.Create(*memprofile)
				if err != nil {
					log.Fatal(err)
				}
				pprof.WriteHeapProfile(mf)
				defer mf.Close()
			}
		}
	}
	// try to handle os interrupt(signal terminated)
	go onKill(c)

	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Pong, but shitty")

	if err := ebiten.RunGame(NewGame()); err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
}
