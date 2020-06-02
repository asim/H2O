package geo

import (
	"math"
)

const (
	earthRadius = 6372.8 // km
)

func deg2Rad(deg float64) float64 {
	return deg * (math.Pi / 180.0)
}

// Haversine uses the haversine formula to return a great-circle distance
// between two latitude/longitude points. Distance returned is in kilometers.
func Haversine(xLat, xLon, yLat, yLon float64) float64 {
	dLat := deg2Rad(yLat - xLat)
	dLon := deg2Rad(yLon - xLon)

	sin := math.Sin
	cos := math.Cos

	a := sin(dLat/2)*sin(dLat/2) + cos(deg2Rad(xLat))*
		cos(deg2Rad(yLat))*sin(dLon/2)*sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

// Returns the distance between two points (given by lat and long coordinates) in meters
func HaversineInMeters(xLat, xLon, yLat, yLon float64) float64 {
	return Haversine(xLat, xLon, yLat, yLon) * 1000.00
}
