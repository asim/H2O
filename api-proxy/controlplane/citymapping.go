package controlplane

// we store a static list of cityMaps since this is only for H1 and we have a fixed
// set of cities to work with, plus it's simpler
var hostMap = map[string]string{
	"api-driver-london.elasticride.com":         "LON",
	"api-driver-london-test.elasticride.com":    "LON",
	"api-driver-london-staging.elasticride.com": "LON",

	"api-driver.elasticride.com":         "LON",
	"api-driver-test.elasticride.com":    "LON",
	"api-driver-staging.elasticride.com": "LON",

	"api-driver-dublin.elasticride.com":         "DUB",
	"api-driver-dublin-test.elasticride.com":    "DUB",
	"api-driver-dublin-staging.elasticride.com": "DUB",

	"api-driver-boston.elasticride.com":         "BOS",
	"api-driver-boston-test.elasticride.com":    "BOS",
	"api-driver-boston-staging.elasticride.com": "BOS",

	"api-driver-chicago.elasticride.com":         "CHI",
	"api-driver-chicago-test.elasticride.com":    "CHI",
	"api-driver-chicago-staging.elasticride.com": "CHI",

	"api-driver-nyc.elasticride.com":         "NYC",
	"api-driver-nyc-test.elasticride.com":    "NYC",
	"api-driver-nyc-staging.elasticride.com": "NYC",

	"api-driver-toronto.elasticride.com":         "TOR",
	"api-driver-toronto-test.elasticride.com":    "TOR",
	"api-driver-toronto-staging.elasticride.com": "TOR",

	"api-driver-montreal.elasticride.com":         "MTR",
	"api-driver-montreal-test.elasticride.com":    "MTR",
	"api-driver-montreal-staging.elasticride.com": "MTR",

	"api-driver-madrid.elasticride.com":         "MAD",
	"api-driver-madrid-test.elasticride.com":    "MAD",
	"api-driver-madrid-staging.elasticride.com": "MAD",

	"api-driver-barcelona.elasticride.com":         "BCN",
	"api-driver-barcelona-test.elasticride.com":    "BCN",
	"api-driver-barcelona-staging.elasticride.com": "BCN",

	"api-driver-dc.elasticride.com":         "WAS",
	"api-driver-dc-test.elasticride.com":    "WAS",
	"api-driver-dc-staging.elasticride.com": "WAS",

	"api-driver-washington.elasticride.com":         "WAS",
	"api-driver-washington-test.elasticride.com":    "WAS",
	"api-driver-washington-staging.elasticride.com": "WAS",

	"api-driver-osaka.elasticride.com":         "OSA",
	"api-driver-osaka-test.elasticride.com":    "OSA",
	"api-driver-osaka-staging.elasticride.com": "OSA",

	"api-driver-tokyo.elasticride.com":         "TYO",
	"api-driver-tokyo-test.elasticride.com":    "TYO",
	"api-driver-tokyo-staging.elasticride.com": "TYO",
}
