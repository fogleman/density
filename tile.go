package density

import (
	"image"
	"image/color"
	"math"

	"github.com/lucasb-eyer/go-colorful"
)

const TileSize = 256

func TileXY(zoom int, lat, lng float64) (x, y int) {
	fx, fy := TileFloatXY(zoom, lat, lng)
	x = int(math.Floor(fx))
	y = int(math.Floor(fy))
	return
}

func TileFloatXY(zoom int, lat, lng float64) (x, y float64) {
	lat_rad := lat * math.Pi / 180
	n := math.Pow(2, float64(zoom))
	x = (lng + 180) / 360 * n
	y = (1 - math.Log(math.Tan(lat_rad)+(1/math.Cos(lat_rad)))/math.Pi) / 2 * n
	return
}

func TileLatLng(zoom, x, y int) (lat, lng float64) {
	n := math.Pow(2, float64(zoom))
	lng = float64(x)/n*360 - 180
	lat = math.Atan(math.Sinh(math.Pi*(1-2*float64(y)/n))) * 180 / math.Pi
	return
}

type IntPoint struct {
	X, Y int
}

type Tile struct {
	Zoom, X, Y int
	Grid       map[IntPoint]float64
	Points     int
}

func NewTile(zoom, x, y int) *Tile {
	grid := make(map[IntPoint]float64)
	return &Tile{zoom, x, y, grid, 0}
}

func (tile *Tile) Add(lat, lng float64) {
	u, v := TileFloatXY(tile.Zoom, lat, lng)
	u -= float64(tile.X)
	v -= float64(tile.Y)
	v = 1 - v
	u *= TileSize
	v *= TileSize
	x := int(math.Floor(u))
	y := int(math.Floor(v))
	u = u - math.Floor(u)
	v = v - math.Floor(v)
	tile.Grid[IntPoint{x + 0, y + 0}] += (1 - u) * (1 - v)
	tile.Grid[IntPoint{x + 0, y + 1}] += (1 - u) * v
	tile.Grid[IntPoint{x + 1, y + 0}] += u * (1 - v)
	tile.Grid[IntPoint{x + 1, y + 1}] += u * v
	tile.Points++
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
