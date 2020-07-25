package future

import (
	"image"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

var glyphAdvanceCache = map[font.Face]map[rune]fixed.Int26_6{}

func fixed26_6ToFloat64(x fixed.Int26_6) float64 {
	return float64(x>>6) + float64(x&((1<<6)-1))/float64(1<<6)
}

func glyphAdvance(face font.Face, r rune) fixed.Int26_6 {
	m, ok := glyphAdvanceCache[face]
	if !ok {
		m = map[rune]fixed.Int26_6{}
		glyphAdvanceCache[face] = m
	}

	a, ok := m[r]
	if !ok {
		a, _ = face.GlyphAdvance(r)
		m[r] = a
	}

	return a
}

func MeasureString(text string, face font.Face) image.Point {
	var w, h fixed.Int26_6

	m := face.Metrics()
	faceHeight := m.Height
	faceDescent := m.Descent

	fx, fy := fixed.I(0), fixed.I(0)
	prevR := rune(-1)

	runes := []rune(text)

	for _, r := range runes {
		if prevR >= 0 {
			fx += face.Kern(prevR, r)
		}
		if r == '\n' {
			fx = fixed.I(0)
			fy += faceHeight
			prevR = rune(-1)
			continue
		}

		fx += glyphAdvance(face, r)

		if fx > w {
			w = fx
		}
		if (fy + faceHeight) > h {
			h = fy + faceHeight
		}

		prevR = r
	}

	bounds := image.Point{
		X: int(math.Ceil(fixed26_6ToFloat64(w))),
		Y: int(math.Ceil(fixed26_6ToFloat64(h + faceDescent))),
	}

	return bounds
}
