# High-density Point Maps

Render millions of points on a map.

![Map](http://i.imgur.com/qhkUlAK.png)

### Demo Site

This demo shows 77 million taxi pickups in NYC - from January to June 2015.

https://www.michaelfogleman.com/static/density/

### Dependencies

  - Go
  - Cassandra

### Download

    go get github.com/fogleman/density

### Loading Data

Cassandra is used to store large amounts of data. Data is clustered by `(zoom, x, y)` so
that all of the points inside of a tile can quickly be fetched for rendering, even
faster than PostGIS with an index.

First, create a new keyspace and table to house the data.

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

Next, load points into the database from a CSV file using the loader script.

    go run loader/main.go < input.csv

Several command line options are available:

| Flag | Default | Description |
| --- | --- | --- |
| -keyspace | density | Cassandra keyspace to load into |
| -table | points | Cassandra table to load into |
| -lat | 0 | CSV column index of latitude values |
| -lng | 1 | CSV column index of longitude values |
| -zoom | 18 | Zoom level to use for binning points |

Just run the loader whenever you need to insert more data.

### Serving Tiles

Once the data is loaded into Cassandra, the tile server can be run for rendering tiles on the fly.

    go run server/main.go

| Flag | Default | Description |
| --- | --- | --- |
| -keyspace | density | Cassandra keyspace to load from |
| -table | points | Cassandra table to load from |
| -zoom | 18 | Zoom level that was used for binning points |
| -port | 5000 | Tile server port number |
| -cache | cache | Directory for caching tile images |

### Serving Maps

A simple Leaflet map is provided to display a base map with the point tiles on top in a separate layer.

    cd web
    python -m SimpleHTTPServer

Then visit [http://localhost:8000/](http://localhost:8000/) in your browser!

### TODO

- tile rendering options
- multiple layers
