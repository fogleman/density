package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/fogleman/density"
	"github.com/gocql/gocql"
)

/*
create keyspace density
    with replication = {
        'class': 'SimpleStrategy',
        'replication_factor': 1
    }
    and durable_writes = false;

create table points (
    zoom int,
    x int,
    y int,
    lat double,
    lng double,
    primary key ((zoom, x, y), lat, lng)
);
*/

const CqlHost = "127.0.0.1"
const Workers = 64

var Keyspace string
var Table string
var LatIndex int
var LngIndex int
var Zoom int
var Query string

func init() {
	flag.StringVar(&Keyspace, "keyspace", "density", "keyspace name")
	flag.StringVar(&Table, "table", "points", "table name")
	flag.IntVar(&LatIndex, "lat", 0, "column index for latitude")
	flag.IntVar(&LngIndex, "lng", 1, "column index for longitude")
	flag.IntVar(&Zoom, "zoom", 18, "tile zoom")
}

type Point struct {
	Lat, Lng float64
}

func insert(session *gocql.Session, lat, lng float64) {
	x, y := density.TileXY(Zoom, lat, lng)
	if err := session.Query(Query, Zoom, x, y, lat, lng).Exec(); err != nil {
		log.Fatal(err)
	}
}

func worker(session *gocql.Session, points <-chan Point) {
	for point := range points {
		insert(session, point.Lat, point.Lng)
	}
}

func main() {
	flag.Parse()

	Query = "INSERT INTO %s (zoom, x, y, lat, lng) VALUES (?, ?, ?, ?, ?);"
	Query = fmt.Sprintf(Query, Table)

	cluster := gocql.NewCluster(CqlHost)
	cluster.Keyspace = Keyspace
	session, _ := cluster.CreateSession()
	defer session.Close()

	points := make(chan Point, 1024)
	for i := 0; i < Workers; i++ {
		go worker(session, points)
	}

	reader := csv.NewReader(os.Stdin)
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		lat, _ := strconv.ParseFloat(record[LatIndex], 64)
		lng, _ := strconv.ParseFloat(record[LngIndex], 64)
		if lat == 0 || lng == 0 {
			continue
		}
		if lat < -90 || lat > 90 {
			continue
		}
		if lng < -180 || lng > 180 {
			continue
		}
		points <- Point{lat, lng}
	}
	close(points)
}
