package hob

import (
	"fmt"
	"time"
)

type Country struct {
	Cctld      string `json:"cctld"`
	ISO_3166_1 string `json:"iso_3166_1"`
}

type Centroid struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type GeoInfo struct {
	Centroid Centroid `json:"centroid"`
	Minimum  Location `json:"minimum"`
	Maximum  Location `json:"maximum"`
}

type Phone struct {
	CallingCode string `json:"callingCode"`
	TrunkPrefix string `json:"trunkPrefix"`
}

type H4B struct {
	RestrictDrivers bool `json:"restrictDrivers"`
}

type Prebook struct {
	Enabled             bool  `json:"enabled"`
	ShowDestination     bool  `json:"showDestination"`
	ShowPrice           bool  `json:"showPrice"`
	PollInterval        int32 `json:"pollInterval"`
	StartupPollInterval int32 `json:"startupPollInterval"`
	ShowFilters         bool  `json:"showFilters"`
}

// miscellaneous fields used by https://github.com/HailoOSS/go-api-lib/blob/master/driverconfig/driverconfig.go#L49
// as this is a library not really sure what service to put them, so will stick them here for now.
type Misc struct {
	DriverTermsUrl      string `json:"driverTerms"`   // The URL used in the driver app to point to Hailos' Terms&Conditions
	FallbackOnOsm       bool   `json:"fallbackOnOsm"` // The flag used in the driver app to indicate whether to use a fallback to Open Street Maps
	EnableProdDebugMenu bool   `json:"enableProdDebugMenu"`
	DistanceDisplayUnit int32  `json:"distanceDisplayUnit"` // Indicates whether the city uses miles or km in the driver app (MILES_UNIT = 0, KM_UNIT = 1)
	PayWithHailo        bool   `json:"payWithHailo"`
	ShowRouteToPickup   bool   `json:"showRouteToPickup"`
	CancellationPolicy  string `json:"cancellationPolicy"` // The URL to the Hob cancellation policy
	ShowStats           bool   `json:"showStats"`
	ShowAcceptRate      bool   `json:"showAcceptRate"`
	HelpCentreUrl       string `json:"helpCentre"` // The URL used in the driver app to point to Hailo Help pages
}

type Fastpay struct {
	Enabled   bool                        `json:"enabled"`
	Countries map[string][]FastpayCountry `json:"countries"`
}

type FastpayCountry struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// Options to display to the user when booking for this hob
type VehicleOptions struct {
	Accessible bool    `json:"accessible"` // Show accessible on/off switch
	Passengers []int32 `json:"passengers"` // Number of seats available in vehicles. Ex: 5, 6
}

type HobStatus string

const (
	HobStatusBeta     = HobStatus("BETA")
	HobStatusEnabled  = HobStatus("ENABLED")
	HobStatusDisabled = HobStatus("DISABLED")
	HobStatusHidden   = HobStatus("HIDDEN")
)

// Hob represents one of our regulatory areas
type Hob struct {
	Code                   string         `json:"code" readOnly:"true"`
	Name                   string         `json:"name"`
	Status                 HobStatus      `json:"status" enum:"BETA,ENABLED,DISABLED,HIDDEN"`
	Country                Country        `json:"country"`
	Currency               string         `json:"currency"`
	Language               string         `json:"language"`
	GeoInfo                GeoInfo        `json:"geoInfo"`
	Phone                  Phone          `json:"phone"`
	Timezone               string         `json:"timezone"`
	DefaultLocale          string         `json:"defaultLocale"`
	Locale                 string         `json:"locale"`
	Misc                   Misc           `json:"misc"`
	FastestFirst           bool           `json:"fastestFirst"` // Specified whether the "fastest first" feature in the passenger app is enabled
	H4B                    H4B            `json:"h4b"`
	Prebook                Prebook        `json:"prebook"`
	VehicleOptions         VehicleOptions `json:"vehicleOptions"`
	JobOfferVolumeOverride bool           `json:"jobOfferVolumeOverride" description:"Allow user to override job offer volume"`
	CustomJobRingtone      bool           `json:"customJobRingtone"`
	Fastpay                Fastpay        `json:"fastpay"`
}

// Location yields a time.Location appropriate for this HOB, or an error if failed to load
func (h *Hob) Location() (*time.Location, error) {
	tz := h.Timezone
	if tz == "" {
		return nil, fmt.Errorf("No timezone defined for HOB '%v'", h.Code)
	}
	l, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("Failed to load timezone '%v' for HOB '%v': %v", tz, h.Code, err)
	}
	return l, nil
}

// LocalTime turns a UTC time.Time into a localised time for this HOB, based on the timezone
// If the HOB does not have a valid timezone defined, we will return the SAME time, and an error
func (h *Hob) LocalTime(t time.Time) (time.Time, error) {
	l, err := h.Location()
	if err != nil {
		return t, err
	}
	return t.In(l), nil
}
