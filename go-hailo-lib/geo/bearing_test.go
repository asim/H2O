package geo

import (
	"math"
	"testing"
)

// Our epsilon value - it's quite inaccurate but good enough for what we have now
const eps = 1.0

type bearingResult struct {
	pLat         float64
	pLng         float64
	qLat         float64
	qLng         float64
	bearing      float64
	finalBearing float64
}

type path struct {
	currentLat            float64
	currentLng            float64
	destinationLat        float64
	destinationLng        float64
	viaLat                float64
	viaLng                float64
	extraDestinationAngle float64
	coneAngle             float64
	inPath                bool
}

var bearingTestData []bearingResult
var paths []path

func init() {
	bearingTestData = []bearingResult{
		// Badhdad to Osaka
		bearingResult{35.0, 45.0, 35.0, 135.0, 60, 120},

		// Madrid to Athens
		bearingResult{40.4167754, -3.7037902, 37.983917, 23.7293599, 88, 105},

		// Paris to Berlin
		bearingResult{48.856614, 2.3522219, 52.52000659999999, 13.404954, 59, 66},
	}

	paths = []path{
		// via points that are in the cone
		path{51.5130, -0.117, 51.490714, -0.270125, 51.4647, -0.282, 50, 28.0, true},
		path{51.5130, -0.117, 51.490714, -0.270125, 51.4839, -0.237, 50, 28.0, true},
		path{51.5130, -0.117, 51.490714, -0.270125, 51.5098, -0.189, 50, 28.0, true},

		// via points that are out of the cone
		path{51.5130, -0.117, 51.490714, -0.270125, 51.5536, -0.214, 50, 28.0, false},
		path{51.5130, -0.117, 51.490714, -0.270125, 51.5700, -0.167, 50, 28.0, false},
		path{51.5130, -0.117, 51.490714, -0.270125, 51.5517, -0.147, 50, 28.0, false},

		// borderline via points - but they do not fall within the cone
		path{51.5130, -0.117, 51.490714, -0.270125, 51.5440, -0.280, 50, 28.0, false},
		path{51.5130, -0.117, 51.490714, -0.270125, 51.4418, -0.253, 50, 28.0, false},
	}
}

func TestBearings(t *testing.T) {
	for i, bt := range bearingTestData {
		bearing, finalBearing := Bearings(bt.pLat, bt.pLng, bt.qLat, bt.qLng)

		if math.Abs(bearing-bt.bearing) > eps {
			t.Errorf("[Test %d] Calculated and expected initial bearings do not match [calculated=%f, expected=%f]", i, bearing, bt.bearing)
		}

		if math.Abs(finalBearing-bt.finalBearing) > eps {
			t.Errorf("[Test %d] Calculated and expected final bearings do not match [calculated=%f, expected=%f]", i, finalBearing, bt.finalBearing)
		}
	}
}

func TestInPath(t *testing.T) {
	for i, p := range paths {
		in := IsInPath(p.currentLat, p.currentLng, p.destinationLat, p.destinationLng, p.viaLat, p.viaLng, p.extraDestinationAngle, p.coneAngle)

		if p.inPath != in {
			t.Errorf("[Test %d] Mismatch for is in path [path=%v, in=%t]", i, p, in)
		}
	}
}
