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
	"os"
	"path"
	"strconv"

	"github.com/fogleman/density"
	"github.com/gocql/gocql"
	"github.com/gorilla/mux"
	"github.com/lucasb-eyer/go-colorful"
)

const CachePath = "cache"

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
	n := 2
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

type Key struct {
	X, Y int
}

func loadPoints(session *gocql.Session, zoom, x, y, tx, ty int, grid map[Key]float64) int {
	lat2, lng1 := density.TileLatLng(zoom, x, y)
	lat1, lng2 := density.TileLatLng(zoom, x+1, y+1)

	iter := session.Query(Query, Zoom, tx, ty).Iter()
	var rows int
	var lat, lng float64
	for iter.Scan(&lat, &lng) {
		kx := int(math.Floor((lng - lng1) / (lng2 - lng1) * 256))
		ky := int(math.Floor((lat - lat1) / (lat2 - lat1) * 256))
		grid[Key{kx, ky}]++
		rows++
	}

	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}

	return rows
}

func getPoints(session *gocql.Session, zoom, x, y int, grid map[Key]float64) int {
	if zoom < 12 {
		return 0
	}
	p := 1 // padding
	var x0, y0, x1, y1 int
	if zoom > Zoom {
		d := int(math.Pow(2, float64(zoom-Zoom)))
		x0, y0 = x/d-p, y/d-p
		x1, y1 = x/d+p, y/d+p
	} else if zoom < Zoom {
		d := int(math.Pow(2, float64(Zoom-zoom)))
		x0, y0 = x*d-p, y*d-p
		x1, y1 = (x+1)*d-1+p, (y+1)*d-1+p
	} else {
		x0, y0 = x-p, y-p
		x1, y1 = x+p, y+p
	}
	var rows int
	for tx := x0; tx <= x1; tx++ {
		for ty := y0; ty <= y1; ty++ {
			rows += loadPoints(session, zoom, x, y, tx, ty, grid)
		}
	}
	return rows
}

func render(grid map[Key]float64, scale float64) (image.Image, bool) {
	im := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	ok := false
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
			t *= 32
			t /= scale
			t /= tw
			t = t / (t + 1)
			a := uint8(255 * math.Pow(t, 0.5))
			c := colorful.Hsv(215.0, 1-t*t, 1)
			r, g, b := c.RGB255()
			im.SetNRGBA(x, 255-y, color.NRGBA{r, g, b, a})
			ok = true
		}
	}
	return im, ok
}

func cachePath(zoom, x, y int) string {
	return fmt.Sprintf("%s/%d/%d/%d.png", CachePath, zoom, x, y)
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zoom := parseInt(vars["zoom"])
	x := parseInt(vars["x"])
	y := parseInt(vars["y"])

	p := cachePath(zoom, x, y)
	if !pathExists(p) {
		// nothing in cache, render the tile
		session, _ := Cluster.CreateSession()
		defer session.Close()
		grid := make(map[Key]float64)
		scale := math.Pow(4, float64(Zoom-zoom))
		rows := getPoints(session, zoom, x, y, grid)
		fmt.Println(zoom, x, y, rows)
		im, ok := render(grid, scale)
		if ok {
			// save tile in cache
			d, _ := path.Split(p)
			os.MkdirAll(d, 0777)
			f, err := os.Create(p)
			if err != nil {
				// unable to cache, just send the png
				w.Header().Set("Content-Type", "image/png")
				png.Encode(w, im)
				return
			}
			png.Encode(f, im)
			f.Close()
		} else {
			// blank tile
			http.NotFound(w, r)
			return
		}
	}
	// serve cached tile
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, p)
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
