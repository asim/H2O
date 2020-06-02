package jobutils

import (
	"testing"

	jobproto "github.com/HailoOSS/job-service/proto"
	"github.com/HailoOSS/protobuf/proto"
)

// Run some tests against the address string builder
func TestBuildAddressString(t *testing.T) {

	testCases := []struct {
		Address *jobproto.Address
		Output  string
	}{
		{
			Address: &jobproto.Address{
				Geocoded: proto.String("Some Road, London"),
				Detail:   proto.String("19"),
			},
			Output: "19 Some Road, London",
		},
		{
			Address: &jobproto.Address{
				Geocoded: proto.String(""),
				Detail:   proto.String("19b"),
			},
			Output: "",
		},
		{
			Address: &jobproto.Address{
				Geocoded: proto.String("Some Street, Dublin"),
			},
			Output: "Some Street, Dublin",
		},
	}

	for _, test := range testCases {
		res := BuildAddressString(test.Address, "LON")

		if res != test.Output {
			t.Errorf("Result '%s' does not match expected '%s'", res, test.Output)
		}
	}
}
