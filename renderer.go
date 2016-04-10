package density

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/lucasb-eyer/go-colorful"
)

type Renderer struct {
	Sources []*TileSource
}

func NewRenderer(sources ...*TileSource) *Renderer {
	return &Renderer{sources}
}

func (r *Renderer) Render(zoom, x, y int) (image.Image, bool) {
	start := time.Now()
	kernel := NewKernel(2)

	n := len(r.Sources)
	tiles := make([]*Tile, n)
	scales := make([]float64, n)
	hues := make([]float64, n)
	points := 0
	for i := range tiles {
		tiles[i] = r.Sources[i].GetTile(zoom, x, y)
		scales[i] = 32 / math.Pow(4, float64(r.Sources[i].BaseZoom-zoom))
		hues[i] = r.Sources[i].Hue
		points += tiles[i].Points
	}

	im := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
	ok := false
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			var total float64
			var hue float64
			for i, tile := range tiles {
				t := tile.Sample(kernel, scales[i], x, y)
				total += t
				hue += t * hues[i]
			}
			if total == 0 {
				continue
			}
			hue /= total
			total /= float64(n)
			t := total
			t = t / (t + 1)
			a := uint8(255 * math.Pow(t, 0.5))
			c := colorful.Hsv(hue, 1-t*t, 1)
			r, g, b := c.RGB255()
			im.SetNRGBA(x, TileSize-1-y, color.NRGBA{r, g, b, a})
			ok = true
		}
	}

	elapsed := time.Now().Sub(start).Seconds()
	if ok {
		fmt.Printf("RENDER (%d %d %d) %8d pts %.3fs\n",
			zoom, x, y, points, elapsed)
	}
	return im, ok
}
