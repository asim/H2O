// +build integration

package timeseries

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/service/config"
)

/*
Test:
 - wide rows (granularity ~ 30 days)    DONE
 - narror rows (granularity ~ 10 mins)  DONE
 - lots of things with same timestamp   DONE
 - large interval betweeen things       DONE
 - slice within a wide row              DONE
 - pagination							DONE
 - reversing							DONE
*/

type TestThing struct {
	Id   string
	Time time.Time
}

func loadConfig() {
	buf := bytes.NewBufferString(`{"hailo": {"service": {"cassandra": {"hosts": ["localhost:19160"]}}}}`)
	config.Load(buf)
}

func TestWideRow(t *testing.T) {
	ts := &TimeSeries{
		Ks:             "testing",
		Cf:             "TestWideRow",
		RowGranularity: time.Hour * 24 * 30,
		Marshaler: func(i interface{}) (uid string, t time.Time) {
			return i.(*TestThing).Id, i.(*TestThing).Time
		},
		IndexCf: "TestWideRowIndex",
	}
	testLoadAndFetch(t, ts, 1)
}

func TestNarrowRow(t *testing.T) {
	ts := &TimeSeries{
		Ks:             "testing",
		Cf:             "TestNarrowRow",
		RowGranularity: time.Second * 10, // zomg - much tiny
		Marshaler: func(i interface{}) (uid string, t time.Time) {
			return i.(*TestThing).Id, i.(*TestThing).Time
		},
		IndexCf: "TestNarrowRowIndex",
	}
	testLoadAndFetch(t, ts, 1)
}

func TestPrimeRow(t *testing.T) {
	ts := &TimeSeries{
		Ks:             "testing",
		Cf:             "TestPrimeRow",
		RowGranularity: time.Minute * 13,
		Marshaler: func(i interface{}) (uid string, t time.Time) {
			return i.(*TestThing).Id, i.(*TestThing).Time
		},
		IndexCf: "TestPrimeRowIndex",
	}
	testLoadAndFetch(t, ts, 1)
}

func TestNoInterval(t *testing.T) {
	ts := &TimeSeries{
		Ks:             "testing",
		Cf:             "TestNoInterval",
		RowGranularity: time.Minute,
		Marshaler: func(i interface{}) (uid string, t time.Time) {
			return i.(*TestThing).Id, i.(*TestThing).Time
		},
		IndexCf: "TestNoIntervalIndex",
	}

	loadInData(t, ts, 0)

	// ---
	// now read them ALL back -- should still work, although we get no guarantees about order

	it, err := ts.UnboundedIterator("")
	if err != nil {
		t.Fatalf("Failed to get iterator: %v", err)
	}

	count := 0
	readMap := make(map[string]int)
	for it.Next() {
		count++
		thing := &TestThing{}
		if err := it.Item().Unmarshal(thing); err != nil {
			t.Errorf("Failed to unmarshal thing %v: %v", count, err)
		}
		if _, ok := readMap[thing.Id]; !ok {
			readMap[thing.Id] = 1
		} else {
			readMap[thing.Id]++
			t.Errorf("Have read back ID '%v' %v times - expecting to read each item once", thing.Id, readMap[thing.Id])
		}
	}

	if count != 10000 {
		t.Errorf("Failed to read %v things back with iterator; got %v", 10000, count)
	}
}

func TestLargeInterval(t *testing.T) {
	ts := &TimeSeries{
		Ks:             "testing",
		Cf:             "TestLargeInterval",
		RowGranularity: time.Minute,
		Marshaler: func(i interface{}) (uid string, t time.Time) {
			return i.(*TestThing).Id, i.(*TestThing).Time
		},
		IndexCf: "TestLargeIntervalIndex",
	}

	loadConfig()

	loadInData(t, ts, 3600) // one hour between each data point, and only a minute row size

	t.Log("Completed 10k data write, reading via unbounded iterator...")

	// ---
	// now read them ALL back
	it, err := ts.UnboundedIterator("")
	if err != nil {
		t.Fatalf("Failed to get iterator: %v", err)
	}

	testReadBack(t, it, 10000, "1", "10000", true)

	t.Log("Completed 10k data read via unbounded iterator, reading via time-slice iterator...")
	// ---

	// now read back a chunk of them, within a time range - should be 200 (time range inclusive)
	start := time.Unix(1387065600, 0)
	end := start.Add(time.Hour * 1999)

	it = ts.Iterator(start, end, "", "")
	testReadBack(t, it, 2000, "1", "2000", true)

	t.Log("Completed 10k data read via normal iterator")
}

func loadInData(t *testing.T, ts *TimeSeries, period int) {
	pool, err := cassandra.ConnectionPool(ts.Ks)
	if err != nil {
		t.Fatalf("C* connection pool error: %v", err)
	}

	currTime := 1387065600 // middle of DEC 2013
	for i := 0; i < 10000; i++ {
		thing := &TestThing{
			Id:   fmt.Sprintf("%v", i+1),
			Time: time.Unix(int64(currTime), 0),
		}
		currTime += period

		w := pool.Writer()
		ts.Map(w, thing, nil)
		if err := w.Run(); err != nil {
			t.Fatalf("Error writing to C*: %v", err)
		}
	}
}

func testLoadAndFetch(t *testing.T, ts *TimeSeries, period int) {
	loadConfig()

	loadInData(t, ts, period)

	t.Log("Completed 10k data write, reading via unbounded iterator...")

	// ---
	// now read them ALL back
	it, err := ts.UnboundedIterator("")
	if err != nil {
		t.Fatalf("Failed to get iterator: %v", err)
	}

	testReadBack(t, it, 10000, "1", "10000", true)

	t.Log("Completed 10k data read via unbounded iterator, reading via time-slice iterator...")
	// ---

	// now read back a chunk of them, within a time range - should be 2k (time range inclusive)

	it = ts.Iterator(time.Unix(1387070926, 0), time.Unix(1387072925, 0), "", "")
	testReadBack(t, it, 2000, "5327", "7326", true)

	t.Log("Completed 10k data read via normal iterator, reading via reversed time-slice iterator...")

	// ---

	// ditto in reverse

	it = ts.ReversedIterator(time.Unix(1387070926, 0), time.Unix(1387072925, 0), "", "")
	testReadBack(t, it, 2000, "7326", "5327", false)

	t.Log("Completed 10k data read via reversed iterator")

	// ---

	// forward and reverse iterator, with pagination (read in batches of N and then supply "lastId" to get next page)
	testPaginationReadBack(t, ts, true)
	testPaginationReadBack(t, ts, false)
}

func testReadBack(t *testing.T, it Iterator, n int, exptFirst, exptLast string, forward bool) {
	var (
		lastT time.Time
		count int
	)
	readMap := make(map[string]int)
	readThings := make([]*TestThing, 0)
	for it.Next() {
		count++
		thing := &TestThing{}
		if err := it.Item().Unmarshal(thing); err != nil {
			t.Errorf("Failed to unmarshal thing %v: %v", count, err)
		}
		if forward {
			if !lastT.IsZero() && thing.Time.Before(lastT) {
				t.Errorf("Expecting things to be in time order, increasing; got %v followed by %v", lastT, thing.Time)
			}
		} else {
			if !lastT.IsZero() && thing.Time.After(lastT) {
				t.Errorf("Expecting things to be in time order, decreasing; got %v followed by %v", lastT, thing.Time)
			}
		}
		lastT = thing.Time
		if _, ok := readMap[thing.Id]; !ok {
			readMap[thing.Id] = 1
		} else {
			readMap[thing.Id]++
			t.Errorf("Have read back ID '%v' %v times - expecting to read each item once", thing.Id, readMap[thing.Id])
		}
		readThings = append(readThings, thing)
	}

	if count != n {
		t.Errorf("Failed to read %v things back with iterator; got %v", n, count)
	}

	if err := it.Err(); err != nil {
		t.Errorf("Iterator readback fail: %v", err)
	}

	if len(readThings) > 0 {
		first := readThings[0]
		last := readThings[len(readThings)-1]
		if first.Id != exptFirst {
			t.Errorf("First thing ID fetched expected to be '%v', got '%v' (ts=%v)", exptFirst, first.Id, first.Time.Unix())
		}
		if last.Id != exptLast {
			t.Errorf("Last thing ID fetched expected to be '%v', got '%v' (ts=%v)", exptLast, last.Id, last.Time.Unix())
		}
	}
}

func testPaginationReadBack(t *testing.T, ts *TimeSeries, forward bool) {
	pageSize := 10
	lastId := ""
	totalCount := 0
	pages := 0
	for {
		var it Iterator
		if forward {
			it = ts.Iterator(time.Unix(1387070926, 0), time.Unix(1387072925, 0), lastId, "")
		} else {
			it = ts.ReversedIterator(time.Unix(1387070926, 0), time.Unix(1387072925, 0), lastId, "")
		}
		pages++

		count := 0
		for it.Next() {
			count++
			totalCount++
			if count >= pageSize {
				break
			}
		}

		// any in batch?
		if count == 0 {
			break
		}

		// page done; set last id for next page
		lastId = it.Last()
	}

	// assert total count is 2k as expected
	if totalCount != 2000 {
		t.Errorf("Paged iterator read %v total in %v pages, expecting 2000 total", totalCount, pages)
	}
}
