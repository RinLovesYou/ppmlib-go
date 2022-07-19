package ppmlib

import (
	"image"
	"image/color"
)

var (
	white = color.RGBA{255, 255, 255, 255}
	black = color.RGBA{0, 0, 0, 255}
	red   = color.RGBA{255, 0, 0, 255}
	blue  = color.RGBA{0, 0, 255, 255}
)

func (f *Frame) GetImage() *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, 256, 192), []color.Color{white, black, red, blue})

	background := white
	if f.PaperColor == PaperColorBlack {
		background = black
	}

	layer1Stroke := black
	if f.Layer1.PenColor == PenColorRed {
		layer1Stroke = red
	}
	if f.Layer1.PenColor == PenColorBlue {
		layer1Stroke = blue
	}
	if f.Layer1.PenColor == PenColorInverted {
		if background == black {
			layer1Stroke = white
		} else {
			layer1Stroke = black
		}
	}

	layer2Stroke := black
	if f.Layer2.PenColor == PenColorRed {
		layer2Stroke = red
	}

	if f.Layer2.PenColor == PenColorBlue {
		layer2Stroke = blue
	}

	if f.Layer2.PenColor == PenColorInverted {
		if background == black {
			layer2Stroke = white
		} else {
			layer2Stroke = black
		}
	}

	for y := 0; y <= 191; y++ {
		for x := 0; x <= 255; x++ {
			img.Set(x, y, background)

			if f.Layer2.Get(x, y) {
				img.Set(x, y, layer2Stroke)
			}

			if f.Layer1.Get(x, y) {
				img.Set(x, y, layer1Stroke)
			}
		}
	}

	return img
}
