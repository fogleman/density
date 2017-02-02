package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/fogleman/density"
)

const (
	lat  = 0
	lng  = 0
	zoom = 3
	w    = 2048
	h    = 2048
)

var urls = []string{
	"http://a.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png",
	"http://localhost:5000/{z}/{x}/{y}.png",
}

func SavePNG(path string, im image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, im)
}

func MergeLayers(layers []*image.NRGBA) *image.NRGBA {
	result := image.NewNRGBA(layers[0].Bounds())
	for _, layer := range layers {
		draw.Draw(result, result.Bounds(), layer, image.ZP, draw.Over)
	}
	return result
}

func main() {
	layers := make([]*image.NRGBA, len(urls))
	for i := range layers {
		im, err := density.Stitch(urls[i], lat, lng, zoom, w, h)
		if err != nil {
			panic(err)
		}
		layers[i] = im
	}
	im := MergeLayers(layers)
	SavePNG("out.png", im)
}
