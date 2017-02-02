package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/fogleman/density"
	"github.com/gorilla/mux"
)

const CqlHost = "127.0.0.1"

var Port int
var CacheDirectory string
var Keyspace string
var Table string
var BaseZoom int

func init() {
	flag.IntVar(&Port, "port", 5000, "server port")
	flag.StringVar(&CacheDirectory, "cache", "cache", "cache directory")
	flag.StringVar(&Keyspace, "keyspace", "density", "keyspace name")
	flag.StringVar(&Table, "table", "points", "table name")
	flag.IntVar(&BaseZoom, "zoom", 18, "tile zoom")
}

func cachePath(zoom, x, y int) string {
	return fmt.Sprintf("%s/%d/%d/%d.png", CacheDirectory, zoom, x, y)
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func parseInt(x string) int {
	value, _ := strconv.ParseInt(x, 0, 0)
	return int(value)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zoom := parseInt(vars["zoom"])
	x := parseInt(vars["x"])
	y := parseInt(vars["y"])
	p := cachePath(zoom, x, y)
	if !pathExists(p) {
		// nothing in cache, render the tile
		renderer := density.NewRenderer(CqlHost, Keyspace, Table, BaseZoom)
		im, ok := renderer.Render(zoom, x, y)
		if ok {
			// save tile in cache
			d, _ := path.Split(p)
			os.MkdirAll(d, 0777)
			f, err := os.Create(p)
			if err != nil {
				// unable to cache, just send the png
				w.Header().Set("Content-Type", "image/png")
				w.Header().Set("Access-Control-Allow-Origin", "*")
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
	} else {
		fmt.Printf("CACHED (%d %d %d)\n", zoom, x, y)
	}
	// serve cached tile
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeFile(w, r, p)
}

func main() {
	flag.Parse()
	router := mux.NewRouter()
	router.HandleFunc("/{zoom:\\d+}/{x:\\d+}/{y:\\d+}.png", Handler)
	addr := fmt.Sprintf(":%d", Port)
	log.Fatal(http.ListenAndServe(addr, router))
}
