package jobutils

import jobproto "github.com/HailoOSS/job-service/proto"

// Produces an address string (including the detail prepended) HOB is
// included in case there needs to be some localisation in future.
func BuildAddressString(addr *jobproto.Address, hob string) string {

	geocoded := addr.GetGeocoded()
	detail := addr.GetDetail()

	if geocoded == "" {
		return ""
	}

	if detail == "" {
		return geocoded
	}

	return detail + " " + geocoded
}
