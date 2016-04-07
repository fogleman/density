package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/fogleman/density"
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/lucasb-eyer/go-colorful"
)

var Port int
var Keyspace string
var Table string
var Zoom int

var Cluster *gocql.ClusterConfig
var Query string

func init() {
	flag.IntVar(&Port, "port", 5000, "server port")
	flag.StringVar(&Keyspace, "keyspace", "density", "keyspace name")
	flag.StringVar(&Table, "table", "points", "table name")
	flag.IntVar(&Zoom, "zoom", 18, "tile zoom")
}

type kernelItem struct {
	dx, dy int
	w      float64
}

var kernel []kernelItem

func init() {
	n := 3
	for dy := -n; dy <= n; dy++ {
		for dx := -n; dx <= n; dx++ {
			d := math.Sqrt(float64(dx*dx + dy*dy))
			w := math.Max(0, 1-d/float64(n))
			w = math.Pow(w, 2)
			kernel = append(kernel, kernelItem{dx, dy, w})
		}
	}
}

func parseInt(x string) int {
	value, _ := strconv.ParseInt(x, 0, 0)
	return int(value)
}

type Point struct {
	X, Y float64
}

func loadPoints(session *gocql.Session, zoom, x, y int) []Point {
	var result []Point
	var lat, lng float64
	iter := session.Query(Query, zoom, x, y).Iter()
	for iter.Scan(&lat, &lng) {
		result = append(result, Point{lng, lat})
	}
	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}
	return result
}

func getPoints(session *gocql.Session, zoom, x, y int) []Point {
	if zoom < 12 {
		return nil
	}
	var x0, y0, x1, y1 int
	p := 1 // padding
	if zoom == Zoom {
		x0, y0 = x-p, y-p
		x1, y1 = x+p, y+p
	}
	if zoom > Zoom {
		d := int(math.Pow(2, float64(zoom-Zoom)))
		x0, y0 = x/d-p, y/d-p
		x1, y1 = x/d+p, y/d+p
	}
	if zoom < Zoom {
		d := int(math.Pow(2, float64(Zoom-zoom)))
		x0, y0 = x*d-p, y*d-p
		x1, y1 = (x+1)*d-1+p, (y+1)*d-1+p
	}
	var result []Point
	for tx := x0; tx <= x1; tx++ {
		for ty := y0; ty <= y1; ty++ {
			result = append(result, loadPoints(session, Zoom, tx, ty)...)
		}
	}
	return result
}

func Render(zoom, x, y int, points []Point) image.Image {
	lat2, lng1 := density.TileLatLng(zoom, x, y)
	lat1, lng2 := density.TileLatLng(zoom, x+1, y+1)
	d := math.Pow(4, float64(Zoom-zoom))

	type Key struct {
		X, Y int
	}

	grid := make(map[Key]float64)
	for _, point := range points {
		x := int(math.Floor((point.X - lng1) / (lng2 - lng1) * 256))
		y := int(math.Floor((point.Y - lat1) / (lat2 - lat1) * 256))
		grid[Key{x, y}]++
	}

	im := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			var t, tw float64
			for _, k := range kernel {
				nx := x + k.dx
				ny := y + k.dy
				t += grid[Key{nx, ny}] * k.w
				tw += k.w
			}
			if t == 0 {
				continue
			}
			t *= 64
			t /= d
			t /= tw
			t = t / (t + 1)
			a := uint8(255 * math.Pow(t, 0.5))
			c := colorful.Hsv(215.0, 1-t*t, 1)
			r, g, b := c.RGB255()
			im.SetNRGBA(x, 255-y, color.NRGBA{r, g, b, a})
		}
	}
	return im
}

func Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zoom := parseInt(vars["zoom"])
	x := parseInt(vars["x"])
	y := parseInt(vars["y"])

	session, _ := Cluster.CreateSession()
	defer session.Close()
	points := getPoints(session, zoom, x, y)
	fmt.Println(zoom, x, y, len(points))

	im := Render(zoom, x, y, points)
	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, im)
}

func main() {
	flag.Parse()

	Query = "SELECT lat, lng FROM %s WHERE zoom = ? AND x = ? AND y = ?;"
	Query = fmt.Sprintf(Query, Table)

	Cluster = gocql.NewCluster("127.0.0.1")
	Cluster.Keyspace = Keyspace

	router := mux.NewRouter()
	router.HandleFunc("/{zoom:\\d+}/{x:\\d+}/{y:\\d+}.png", Handler)
	addr := fmt.Sprintf(":%d", Port)
	log.Fatal(http.ListenAndServe(addr, router))
}