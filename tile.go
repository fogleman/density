package density

import (
	"image"
	"image/color"
	"math"

	"github.com/lucasb-eyer/go-colorful"
)

const TileSize = 256

func TileXY(zoom int, lat, lng float64) (x int, y int) {
	x = int(math.Floor((lng + 180.0) / 360.0 * (math.Exp2(float64(zoom)))))
	y = int(math.Floor((1.0 - math.Log(math.Tan(lat*math.Pi/180.0)+1.0/math.Cos(lat*math.Pi/180.0))/math.Pi) / 2.0 * (math.Exp2(float64(zoom)))))
	return
}

func TileLatLng(zoom, x, y int) (lat, lng float64) {
	n := math.Pi - 2.0*math.Pi*float64(y)/math.Exp2(float64(zoom))
	lat = 180.0 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(-n)))
	lng = float64(x)/math.Exp2(float64(zoom))*360.0 - 180.0
	return
}

type IntPoint struct {
	X, Y int
}

type Tile struct {
	Zoom, X, Y int
	Lat0, Lng0 float64
	Lat1, Lng1 float64
	Grid       map[IntPoint]float64
}

func NewTile(zoom, x, y int) *Tile {
	lat1, lng0 := TileLatLng(zoom, x, y)
	lat0, lng1 := TileLatLng(zoom, x+1, y+1)
	grid := make(map[IntPoint]float64)
	return &Tile{zoom, x, y, lat0, lng0, lat1, lng1, grid}
}

func (tile *Tile) Add(lat, lng float64) {
	// TODO: bilinear interpolation
	x := int(math.Floor((lng - tile.Lng0) / (tile.Lng1 - tile.Lng0) * TileSize))
	y := int(math.Floor((lat - tile.Lat0) / (tile.Lat1 - tile.Lat0) * TileSize))
	tile.Grid[IntPoint{x, y}]++
}

func (tile *Tile) Render(kernel Kernel, scale float64) (image.Image, bool) {
	im := image.NewNRGBA(image.Rect(0, 0, TileSize, TileSize))
	ok := false
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			var t, tw float64
			for _, k := range kernel {
				nx := x + k.Dx
				ny := y + k.Dy
				t += tile.Grid[IntPoint{nx, ny}] * k.Weight
				tw += k.Weight
			}
			if t == 0 {
				continue
			}
			t *= scale
			t /= tw
			t = t / (t + 1)
			a := uint8(255 * math.Pow(t, 0.5))
			c := colorful.Hsv(215.0, 1-t*t, 1)
			r, g, b := c.RGB255()
			im.SetNRGBA(x, TileSize-1-y, color.NRGBA{r, g, b, a})
			ok = true
		}
	}
	return im, ok
}
