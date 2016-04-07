package density

import "math"

func TileXY(zoom int, lat, lng float64) (x int, y int) {
	x = int(math.Floor((lng + 180.0) / 360.0 * (math.Exp2(float64(zoom)))))
	y = int(math.Floor((1.0 - math.Log(math.Tan(lat*math.Pi/180.0)+1.0/math.Cos(lat*math.Pi/180.0))/math.Pi) / 2.0 * (math.Exp2(float64(zoom)))))
	return
}

func TileLatLng(zoom, x, y int) (lat, lng float64) {
	n := math.Pi - 2.0*math.Pi*float64(y)/math.Exp2(float64(zoom))
	lat = 180.0 / math.Pi * math.Atan(0.5*(math.Exp(n)-math.Exp(-n)))
	lng = float64(x)/math.Exp2(float64(zoom))*360.0 - 180.0
	return
}
