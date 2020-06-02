package graphite

import (
	"testing"
	"time"
)

func TestTimeToGraphite(t *testing.T) {
	v := time.Unix(1390326125, 0)
	s := timeToGraphite(v)
	if s != "17:42_20140121" {
		t.Fatalf("Time %v translated to unexpected %v", v, s)
	}
}

func TestMarshalResult(t *testing.T) {
	json := `[{"target": "asPercent(stats.com.HailoOSS.service.test.results.runs.dg2.assert.com-HailoOSS-test-benchmark-noop.success, sumSeries(stats.com.HailoOSS.service.test.results.runs.dg2.assert.com-HailoOSS-test-benchmark-noop.success))", "datapoints": [[null, 1390324150], [null, 1390324160], [100.0, 1390324170], [100.0, 1390324180], [100.0, 1390324190], [100.0, 1390324200], [100.0, 1390324210], [100.0, 1390324220], [100.0, 1390324230], [100.0, 1390324240], [100.0, 1390324250], [100.0, 1390324260], [100.0, 1390324270], [100.0, 1390324280], [100.0, 1390324290], [100.0, 1390324300], [100.0, 1390324310], [100.0, 1390324320], [100.0, 1390324330], [100.0, 1390324340], [100.0, 1390324350], [100.0, 1390324360], [100.0, 1390324370], [100.0, 1390324380], [100.0, 1390324390], [100.0, 1390324400], [100.0, 1390324410], [100.0, 1390324420], [100.0, 1390324430], [100.0, 1390324440], [100.0, 1390324450], [100.0, 1390324460], [100.0, 1390324470], [100.0, 1390324480], [100.0, 1390324490], [100.0, 1390324500], [100.0, 1390324510], [100.0, 1390324520], [100.0, 1390324530], [100.0, 1390324540], [100.0, 1390324550], [100.0, 1390324560], [100.0, 1390324570], [100.0, 1390324580], [100.0, 1390324590], [100.0, 1390324600], [100.0, 1390324610], [100.0, 1390324620]]}]`
	b := []byte(json)

	res, err := unmarshalResult(b)
	if err != nil {
		t.Fatalf("Error unmarshaling: %v", err)
	}

	if len(res.results) != 1 {
		t.Fatalf("Expecting 1 result, got %v", len(res.results))
	}

	s := res.results[0]
	if len(s.DataPoints) != 48 {
		t.Fatalf("Expecting 48 datapoints, got %v", len(s.DataPoints))
	}
	expected := `asPercent(stats.com.HailoOSS.service.test.results.runs.dg2.assert.com-HailoOSS-test-benchmark-noop.success, sumSeries(stats.com.HailoOSS.service.test.results.runs.dg2.assert.com-HailoOSS-test-benchmark-noop.success))`
	if s.Name != expected {
		t.Fatalf("Expecting target name '%v', got '%v'", expected, s.Name)
	}
}
