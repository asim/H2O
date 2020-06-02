package timeseries

import (
	"fmt"
	"strconv"
	"time"

	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/gossie/src/gossie"
)

const (
	indexBatch = 100
)

var (
	placeholderValue []byte        = []byte("1")
	minGranularity   time.Duration = time.Hour * 24
)

type index struct {
	secondaryIndex string
	ts             *TimeSeries
	values         map[time.Time]bool
	start, end     time.Time
}

func newIndex(ts *TimeSeries, si string) *index {
	return &index{
		ts:             ts,
		secondaryIndex: si,
	}
}

// write will add a row into a mutation to update the index view
func (i *index) write(writer gossie.Writer, rowTime time.Time) {
	if rowTime.IsZero() {
		return
	}
	// truncate to our minimum granularity (24 hours - in order to constrain max cols per row to a few thousand likely)
	rowTime = i.truncate(rowTime)

	row := &gossie.Row{
		Key: i.rowKey(),
		Columns: []*gossie.Column{
			{
				Name:  i.colName(rowTime),
				Value: placeholderValue,
			},
		},
	}
	writer.Insert(i.ts.IndexCf, row)
	//log.Debugf("Writing timeseries INDEX mutation for row key=%v %v", string(i.rowKey()), row)
}

// skip will test if we know about some rowTime (or should simply not bother looking) based on index
// will lazy-load entire index the first time - an error occurs if we fail to load index
func (i *index) skip(rowTime time.Time) (bool, error) {
	if i.ts.IndexCf == "" {
		return false, nil
	}
	if i.values == nil {
		if err := i.load(); err != nil {
			return false, err
		}
	}

	// remember that default value is false
	return !i.values[i.truncate(rowTime)], nil
}

// bounds will find start/end time bounds, lazy-loading entire index first time
// an error happens if we fail to load index
func (i *index) bounds() (start, end time.Time, err error) {
	if i.values == nil {
		err = i.load()
	}
	start, end = i.start, i.end
	return
}

// ---

func (i *index) rowKey() []byte {
	return []byte(fmt.Sprintf("i§%s", i.secondaryIndex))
}

func (i *index) colName(t time.Time) []byte {
	return []byte(fmt.Sprintf("%v", t.Unix()))
}

func (i *index) fromColName(b []byte) (time.Time, error) {
	unix, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(unix, 0), nil
}

// truncate will truncate a time to the granularity we need for the index
func (i *index) truncate(t time.Time) time.Time {
	return t.Truncate(i.granularity())
}

// granularity returns the granularity to use for truncation
func (i *index) granularity() time.Duration {
	if i.ts.RowGranularity < minGranularity {
		return minGranularity
	}
	return i.ts.RowGranularity
}

func (i *index) load() error {
	pool, err := cassandra.ConnectionPool(i.ts.Ks)
	if err != nil {
		return fmt.Errorf("Failed to get C* connection pool: %v", err)
	}

	finished := false
	lastCol := []byte{}
	i.values = make(map[time.Time]bool)

	for !finished {
		finished = true // we'll set back to false if we find any results
		row, err := pool.Reader().Cf(i.ts.IndexCf).Slice(&gossie.Slice{
			Start:    lastCol,
			End:      []byte{},
			Count:    indexBatch,
			Reversed: false,
		}).Get(i.rowKey())
		if err != nil {
			return fmt.Errorf("Error reading from C*: %v", err)
		}
		if row != nil && len(row.Columns) > 0 {
			// just got one column which was last one in previous call, so we are finished
			if len(row.Columns) == 1 && string(row.Columns[0].Name) == string(lastCol) {
				break
			}

			// we found more rows, so not finished
			finished = false

			// add any returned columns to our buffer
			for _, c := range row.Columns {
				// set the lastCol for the next fetch
				lastCol = c.Name
				t, err := i.fromColName(c.Name)
				if err != nil {
					return fmt.Errorf("Conversion fail reading index: %v", err)
				}
				// fill index values
				i.values[t] = true
				// set start, if not set
				if i.start.IsZero() {
					i.start = t
				}
				// set end, always
				i.end = t
			}
		}
	}

	// extend end by truncation value
	i.end = i.end.Add(i.granularity())

	return nil
}
