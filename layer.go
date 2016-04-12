package density

import (
	"image"
	"image/draw"
	"math"
	"net/http"
	"strconv"
	"strings"
)

func Stitch(urlTemplate string, lat, lng float64, zoom, w, h int) (*image.NRGBA, error) {
	layer := NewTileLayer(urlTemplate)
	return layer.GetTiles(lat, lng, zoom, w, h)
}

func GetImage(url string) (image.Image, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	im, _, err := image.Decode(response.Body)
	return im, err
}

type TileLayer struct {
	URLTemplate string
}

func NewTileLayer(urlTemplate string) *TileLayer {
	return &TileLayer{urlTemplate}
}

func (layer *TileLayer) GetTile(z, x, y int) (image.Image, error) {
	url := layer.URLTemplate
	url = strings.Replace(url, "{z}", strconv.Itoa(z), -1)
	url = strings.Replace(url, "{x}", strconv.Itoa(x), -1)
	url = strings.Replace(url, "{y}", strconv.Itoa(y), -1)
	return GetImage(url)
}

func (layer *TileLayer) GetTiles(lat, lng float64, zoom, w, h int) (*image.NRGBA, error) {
	im := image.NewNRGBA(image.Rect(0, 0, w, h))
	cx, cy := TileFloatXY(zoom, lat, lng)
	x0 := cx - float64(w)/2/TileSize
	y0 := cy - float64(h)/2/TileSize
	x1 := cx + float64(w)/2/TileSize
	y1 := cy + float64(h)/2/TileSize
	x0i := int(math.Floor(x0))
	y0i := int(math.Floor(y0))
	x1i := int(math.Floor(x1))
	y1i := int(math.Floor(y1))
	ch := make(chan error)
	for x := x0i; x <= x1i; x++ {
		for y := y0i; y <= y1i; y++ {
			px := int(float64(w) * (float64(x) - x0) / (x1 - x0))
			py := int(float64(h) * (float64(y) - y0) / (y1 - y0))
			go layer.getTilesWorker(im, zoom, x, y, px, py, ch)
		}
	}
	for x := x0i; x <= x1i; x++ {
		for y := y0i; y <= y1i; y++ {
			if err := <-ch; err != nil {
				return nil, err
			}
		}
	}
	return im, nil
}

func (layer *TileLayer) getTilesWorker(im *image.NRGBA, zoom, x, y, px, py int, ch chan error) {
	t, err := layer.GetTile(zoom, x, y)
	if err != nil {
		ch <- err
	}
	draw.Draw(im, image.Rect(px, py, px+TileSize, py+TileSize), t, image.ZP, draw.Src)
	ch <- nil
}
