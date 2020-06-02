package geo

import (
	"testing"
)

type haversineResults struct {
	xLat float64
	xLon float64
	yLat float64
	yLon float64
	km   float64
}

var testData []haversineResults

func init() {
	testData = make([]haversineResults, 10)
	testData[0] = haversineResults{51.5116261, -0.117565, 51.5073, -0.12755, 0.842207}
	testData[1] = haversineResults{51.5116261, -0.117565, 51.501364, -0.14189, 2.034396}
	testData[2] = haversineResults{51.5116261, -0.117565, 51.878670, -0.42002, 45.841919}
	testData[3] = haversineResults{51.5116261, -0.117565, 51.555575, -0.17454, 6.279723}
	testData[4] = haversineResults{51.5116261, -0.117565, 51.501364, -0.14189, 2.034396}
	testData[5] = haversineResults{40.723384, -74.001704, 40.789142, -73.1349, 73.396076}
	testData[6] = haversineResults{40.723384, -74.001704, 40.058323, -74.4056612, 81.504235}
	testData[7] = haversineResults{40.723384, -74.001704, 40.45107, -73.58931, 46.160282}
	testData[8] = haversineResults{40.723384, -74.001704, 40.455568, -73.584860, 46.118976}
	testData[9] = haversineResults{40.723384, -74.001704, 40.464230, -73.573048, 46.277094}
}

func TestHaversine(t *testing.T) {
	for _, d := range testData {
		km := Haversine(d.xLat, d.xLon, d.yLat, d.yLon)
		// We just check approximate accuracy.
		if (km - d.km) > 0.000001 {
			t.Errorf("Unexpected response: got %+v km, Expected %+v km", km, d.km)
		}
	}
}

func TestHaversineInMeters(t *testing.T) {
	for _, d := range testData {
		meters := HaversineInMeters(d.xLat, d.xLon, d.yLat, d.yLon)
		if (meters - d.km*1000.00) > 0.001 {
			t.Errorf("Unexpected response: got %+v m, Expected %+v m", meters, d.km*1000.00)
		}
	}
}
