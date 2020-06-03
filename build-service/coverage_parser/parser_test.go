package coverage_parser

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hailo-platform/H2O/build-service/models"
)

func TestParseLine(t *testing.T) {
	testCases := []struct {
		line     string
		expected models.Coverage
	}{
		{
			line: `ok  	_/ebs/jenkins/jobs/hailo-platform/H2O-build-service-23b9ac1515dd/workspace	0.007s	coverage: 36.1% of statements`,
			expected: models.Coverage{PackageName: "main", Percentage: 36.1},
		},
		{
			line: `ok  	_/ebs/jenkins/jobs/hailo-platform/H2O-build-service-23b9ac1515dd/workspace/validate	0.004s	coverage: 95.8% of statements`,
			expected: models.Coverage{PackageName: "validate", Percentage: 95.8},
		},
	}

	for i, tc := range testCases {
		coverage, err := parseLine(tc.line)
		if err != nil {
			t.Errorf("%v (%d)", err, i)
			continue
		}
		if coverage != tc.expected {
			t.Errorf("Test Case %d Failed", i)
			t.Error("Expected")
			t.Errorf("%+v", tc.expected)
			t.Error("Found")
			t.Errorf("%+v", coverage)
		}
	}
}

func TestGetCoverage(t *testing.T) {
	r := strings.NewReader(testOutput)
	c, err := getCoverage(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != 2 {
		t.Errorf("Expected 2 coverage results, found: %d", len(c))
	}
}

func TestWriteCoverage(t *testing.T) {
	coverage := []models.Coverage{
		{PackageName: "one", Percentage: 10.01},
		{PackageName: "two", Percentage: 99.99},
	}

	out := new(bytes.Buffer)
	writeCoverage(out, coverage)
	expected := `{"one":10.01,"two":99.99}`
	actual := strings.TrimSpace(out.String())
	if actual != expected {
		t.Errorf("Expected: %v, Got: %v", expected, actual)
	}
}

var testOutput = `Testing (normal)
=== RUN TestValidateBuild
--- PASS: TestValidateBuild (0.00 seconds)
=== RUN TestGetNames
--- PASS: TestGetNames (0.00 seconds)
=== RUN TestCreateBuild
--- PASS: TestCreateBuild (0.00 seconds)
=== RUN TestGetBuilds
--- PASS: TestGetBuilds (0.00 seconds)
=== RUN TestCoverage
--- PASS: TestCoverage (0.00 seconds)
=== RUN TestDeleteBuild
--- PASS: TestDeleteBuild (0.00 seconds)
=== RUN TestMissingServiceName
--- PASS: TestMissingServiceName (0.00 seconds)
PASS
coverage: 36.1% of statements
ok  	_/ebs/jenkins/jobs/hailo-platform/H2O-build-service-23b9ac1515dd/workspace	0.007s	coverage: 36.1% of statements
?   	_/ebs/jenkins/jobs/hailo-platform/H2O-build-service-23b9ac1515dd/workspace/models	[no test files]
=== RUN TestBlank
--- PASS: TestBlank (0.00 seconds)
PASS
coverage: 95.8% of statements
ok  	_/ebs/jenkins/jobs/hailo-platform/H2O-build-service-23b9ac1515dd/workspace/validate	0.004s	coverage: 95.8% of statements`
