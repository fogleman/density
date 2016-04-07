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

func insert(session *gocql.Session, lat, lng float64) {
	x, y := density.TileXY(Zoom, lat, lng)
	if err := session.Query(Query, Zoom, x, y, lat, lng).Exec(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.Parse()

	Query = "INSERT INTO %s (zoom, x, y, lat, lng) VALUES (?, ?, ?, ?, ?);"
	Query = fmt.Sprintf(Query, Table)

	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = Keyspace
	session, _ := cluster.CreateSession()
	defer session.Close()

	reader := csv.NewReader(os.Stdin)
	reader.Read() // read header line
	rows := 0
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
		insert(session, lat, lng)
		rows++
		if rows%10000 == 0 {
			fmt.Println(rows)
		}
	}
}
