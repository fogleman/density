package density

import (
	"fmt"
	"log"
	"math"

	"github.com/gocql/gocql"
)

type TileSource struct {
	Cluster  *gocql.ClusterConfig
	Query    string
	BaseZoom int
	Hue      float64
}

func NewTileSource(host, keyspace, table string, baseZoom int, hue float64) *TileSource {
	cluster := gocql.NewCluster(host)
	cluster.Keyspace = keyspace
	query := "SELECT lat, lng FROM %s WHERE zoom = ? AND x = ? AND y = ?;"
	query = fmt.Sprintf(query, table)
	return &TileSource{cluster, query, baseZoom, hue}
}

func (s *TileSource) GetTile(zoom, x, y int) *Tile {
	session, _ := s.Cluster.CreateSession()
	defer session.Close()
	return s.loadTile(session, zoom, x, y)
}

func (s *TileSource) loadPoints(session *gocql.Session, x, y int, tile *Tile) {
	iter := session.Query(s.Query, s.BaseZoom, x, y).Iter()
	var lat, lng float64
	for iter.Scan(&lat, &lng) {
		tile.Add(lat, lng)
	}
	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}
}

func (s *TileSource) loadTile(session *gocql.Session, zoom, x, y int) *Tile {
	tile := NewTile(zoom, x, y)
	if zoom < 12 {
		return tile
	}
	p := 1 // padding
	var x0, y0, x1, y1 int
	if zoom > s.BaseZoom {
		d := int(math.Pow(2, float64(zoom-s.BaseZoom)))
		x0, y0 = x/d-p, y/d-p
		x1, y1 = x/d+p, y/d+p
	} else if zoom < s.BaseZoom {
		d := int(math.Pow(2, float64(s.BaseZoom-zoom)))
		x0, y0 = x*d-p, y*d-p
		x1, y1 = (x+1)*d-1+p, (y+1)*d-1+p
	} else {
		x0, y0 = x-p, y-p
		x1, y1 = x+p, y+p
	}
	for tx := x0; tx <= x1; tx++ {
		for ty := y0; ty <= y1; ty++ {
			s.loadPoints(session, tx, ty, tile)
		}
	}
	return tile
}
