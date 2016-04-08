package density

import (
	"fmt"
	"image"
	"log"
	"math"

	"github.com/gocql/gocql"
)

type Renderer struct {
	Cluster  *gocql.ClusterConfig
	Query    string
	BaseZoom int
}

func NewRenderer(host, keyspace, table string, baseZoom int) *Renderer {
	cluster := gocql.NewCluster(host)
	cluster.Keyspace = keyspace
	query := "SELECT lat, lng FROM %s WHERE zoom = ? AND x = ? AND y = ?;"
	query = fmt.Sprintf(query, table)
	return &Renderer{cluster, query, baseZoom}
}

func (r *Renderer) Render(zoom, x, y int) (image.Image, bool) {
	session, _ := r.Cluster.CreateSession()
	defer session.Close()
	tile := r.loadTile(session, zoom, x, y)
	kernel := NewKernel(2)
	scale := 32 / math.Pow(4, float64(r.BaseZoom-zoom))
	return tile.Render(kernel, scale)
}

func (r *Renderer) loadPoints(session *gocql.Session, x, y int, tile *Tile) {
	iter := session.Query(r.Query, r.BaseZoom, x, y).Iter()
	var lat, lng float64
	for iter.Scan(&lat, &lng) {
		tile.Add(lat, lng)
	}
	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}
}

func (r *Renderer) loadTile(session *gocql.Session, zoom, x, y int) *Tile {
	tile := NewTile(zoom, x, y)
	if zoom < 12 {
		return tile
	}
	p := 1 // padding
	var x0, y0, x1, y1 int
	if zoom > r.BaseZoom {
		d := int(math.Pow(2, float64(zoom-r.BaseZoom)))
		x0, y0 = x/d-p, y/d-p
		x1, y1 = x/d+p, y/d+p
	} else if zoom < r.BaseZoom {
		d := int(math.Pow(2, float64(r.BaseZoom-zoom)))
		x0, y0 = x*d-p, y*d-p
		x1, y1 = (x+1)*d-1+p, (y+1)*d-1+p
	} else {
		x0, y0 = x-p, y-p
		x1, y1 = x+p, y+p
	}
	for tx := x0; tx <= x1; tx++ {
		for ty := y0; ty <= y1; ty++ {
			r.loadPoints(session, tx, ty, tile)
		}
	}
	return tile
}
