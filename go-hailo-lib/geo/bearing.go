package geo

import (
	"math"
)

func radiantsToDegrees(rad float64) float64 {
	return rad * 180.0 / math.Pi
}

// Precalculated angles to speed up the process.
// This is to build the base of the cone based on how much further
// do we allow the the geo asset at its current position to go beyond its destination to reach the via point
var PreComputedExtraAngles map[int]float64 = map[int]float64{
	100: 120.0,
	75:  88.0,
	50:  56.0,
	25:  28.0,
	20:  22.0,
	10:  12.0,
	0:   180.0,
}

// Bearing calculates the initial and final angles (in degrees) taken on the route (P, Q) using Great Circle navigation.
// All latitudes and longitudes are expected to be in degrees.
// The current method is most suitable for short distances.
func Bearings(pLat, pLng, qLat, qLng float64) (float64, float64) {
	ib := initialBearing(pLat, pLng, qLat, qLng)

	fb := initialBearing(qLat, qLng, pLat, pLng)
	// Final bearing is initial (bearing + 180) % 360
	fb = math.Mod(fb+180.0, 360.0)

	return ib, fb
}

func initialBearing(pLat, pLng, qLat, qLng float64) float64 {
	lngDiff := deg2Rad(qLng - pLng)
	// our lat and lng are all in degrees - we need to convert them in radiants
	pLat, pLng, qLat, qLng = deg2Rad(pLat), deg2Rad(pLng), deg2Rad(qLat), deg2Rad(pLng)

	y := math.Sin(lngDiff) * math.Cos(qLat)
	x := math.Cos(pLat)*math.Sin(qLat) - math.Sin(pLat)*math.Cos(qLat)*math.Cos(lngDiff)
	atan := math.Atan2(y, x)

	// Convert result back to degrees and make sure it is positive
	degrees := radiantsToDegrees(atan) + 360.0

	// Now ensure that the result is in the [0,360] range
	return math.Mod(degrees, 360.0)
}

// isInPath determines whether the location via which we are going through is in
// the cone formed by the current position and the destination
func IsInPath(currentLat, currentLng, destinationLat, destinationLng, viaLat, viaLng, extraDestinationAngle, coneAngle float64) bool {
	oppositeAngle := 360.0 - coneAngle

	// Calculate angles
	destinationAngle, _ := Bearings(currentLat, currentLng, destinationLat, destinationLng)
	viaAngle, _ := Bearings(currentLat, currentLng, viaLat, viaLng)

	// Now calculate distances - we use haversine for this.
	// Although not as precise as other methods, it works very well for short distances
	destinationDistance := HaversineInMeters(currentLat, currentLng, destinationLat, destinationLng)
	viaDestinationDistance := HaversineInMeters(viaLat, viaLng, destinationLat, destinationLng)
	viaDistance := HaversineInMeters(currentLat, currentLng, viaLat, viaLng)

	// We apply a reduction on first distance to reduce the radius of the circle which has destination coordinates as the center
	destDistanceReduced := destinationDistance * (extraDestinationAngle / 100.00)

	// Now calculate the difference between the angles around the current point's coordinate
	diffAngles := math.Abs(destinationAngle - viaAngle)

	// is the angle less than our accepted cone angle or is it greater than or equal to the complementary angle
	anglesPredicate := diffAngles <= coneAngle || diffAngles >= oppositeAngle

	// The via point must not be further than the destination; moreover the via point must be reasonably close to the destination point
	distancePredicate := viaDistance <= destinationDistance || viaDestinationDistance <= destDistanceReduced

	return anglesPredicate && distancePredicate
}
