package timeseries

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/gossie/src/gossie"
)

// Item represents something fetched from timeseries
type Item struct {
	id string
	t  time.Time
	b  []byte
}

// Unmarshal will unmarshal the fetched item into a struct
func (i *Item) Unmarshal(into interface{}) error {
	return json.Unmarshal(i.b, into)
}

// Iterator represents a slice of our time series that can be looped through, loading on-demand
type Iterator interface {
	Next() bool
	Item() *Item
	Err() error
	Rewind()
	Last() string
}

type itemIterator struct {
	ts    *TimeSeries
	index *index
	// range definition
	start, end             time.Time
	fromId, secondaryIndex string
	// current buffer
	buffer         []*Item
	bufferI        int
	startCol       []byte
	lastFetchedCol []byte // the last column name added to the buffer
	fetchTime      time.Time
	endOfRange     bool
	lastId         string
	// currently fetched item/err
	currItem *Item
	currErr  error
	// configurable settings
	BatchSize int
	Reverse   bool
}

// NextItem can be called inside `for` and will populate the responses to Err() and Item()
// Responds with true if a new Item() was found (hence you can call Item() to get it)
func (i *itemIterator) Next() bool {
	i.currItem, i.currErr = nil, nil
	if (i.buffer == nil || i.bufferI >= len(i.buffer)) && !i.endOfRange {
		if err := i.fillBuffer(); err != nil {
			i.currErr = err
			return false
		}
	}

	if i.bufferI < len(i.buffer) {
		i.currItem = i.buffer[i.bufferI]
		i.bufferI++
		i.lastId = string(itemToColumnName(i.currItem))
		return true
	}

	return false
}

// Item yields the current item
func (i *itemIterator) Item() *Item {
	return i.currItem
}

// Item yields the current err
func (i *itemIterator) Err() error {
	return i.currErr
}

// Rewind resets this iterator back to the first fetch - and will cause the data
// to be re-fetched from C*
func (i *itemIterator) Rewind() {
	i.buffer = nil
}

// Last returns the lastId fetched (for use with pagination) or "" if none fetched
func (i *itemIterator) Last() string {
	return i.lastId
}

func (i *itemIterator) fillBuffer() error {
	if i.buffer == nil {
		if err := i.initBuffer(); err != nil {
			return err
		}
	}

	// sanity check
	if i.BatchSize < 2 {
		panic("Invalid BatchSize - must be >= 2")
	}

	pool, err := cassandra.ConnectionPool(i.ts.Ks)
	if err != nil {
		return fmt.Errorf("Failed to get C* connection pool: %v", err)
	}

	// init buffer for THIS fetch
	i.buffer = i.buffer[0:0]
	i.bufferI = 0

	// loop rows
	for {
		// we may wish to skip fetching from a row, based on our "index" (which keeps track of which rows have ANY data for efficiency)
		shouldSkip, err := i.index.skip(i.fetchTime)
		if err != nil {
			log.Warnf("Error loading index for timeseries, will proceed trying all rows: %v", err)
		}
		if !shouldSkip {
			// loop chunks of columns within this one row
			for {
				row, err := pool.Reader().Cf(i.ts.Cf).Slice(&gossie.Slice{
					Start:    i.startCol,
					End:      []byte{},
					Count:    i.BatchSize + 1,
					Reversed: i.Reverse,
				}).Get(i.ts.toRowKey(i.secondaryIndex, i.fetchTime))
				if err != nil {
					return fmt.Errorf("Error reading from C*: %v", err)
				}

				// strip off first column if we have already read it, or if it is the "lastId" column as part of a paginated fetch
				if row != nil && len(row.Columns) > 0 {
					if bytes.Equal(row.Columns[0].Name, i.lastFetchedCol) || bytes.Equal(row.Columns[0].Name, []byte(i.fromId)) {
						row.Columns = row.Columns[1:]
					}
				}

				if row == nil || len(row.Columns) == 0 {
					// nothing left in row -- go to the next row
					break
				}

				// add any returned columns to our buffer
				for _, c := range row.Columns {
					// set the startCol for the next fetch
					i.startCol = c.Name
					// column within our specified end time?
					_, ct, err := columnNameToTimeId(c.Name)
					if err != nil {
						return fmt.Errorf("Bad column name found: %v", err)
					}
					if i.isBeyondEndOfRange(ct, false) {
						// end of range
						i.endOfRange = true
						// @todo possibly return an error if no items here
						return nil
					}

					// ok - we'll have this column
					item, err := colToItem(c)
					if err != nil {
						return fmt.Errorf("Bad column found: %v", err)
					}
					i.buffer = append(i.buffer, item)
					i.lastFetchedCol = c.Name

					// got enough now?
					if len(i.buffer) >= i.BatchSize {
						return nil
					}
				}
			}
		}

		// go to next row -- if we're within timing
		i.startCol = []byte{} // reset startCol since we always want to start from beggining of row
		if i.Reverse {
			i.fetchTime = i.fetchTime.Add(-i.ts.RowGranularity)
		} else {
			i.fetchTime = i.fetchTime.Add(i.ts.RowGranularity)
		}
		if i.isBeyondEndOfRange(i.fetchTime, true) {
			// end of range
			i.endOfRange = true
			return nil
		}
	}

	// no return - unreachable
}

// isBeyondEndOfRange tests if a time t is beyond the end of the range, which will depend
// on if we're going forward or backward through things
// the "bucketTest" flag is needed because if going backwards, we need to look at a bucket
// which _starts_ at a time _before_ the start of the range
func (i *itemIterator) isBeyondEndOfRange(t time.Time, bucketTest bool) bool {
	if i.Reverse {
		if bucketTest {
			return t.Add(i.ts.RowGranularity).Before(i.end)
		} else {
			return t.Before(i.end)

		}
	}
	return t.After(i.end)
}

func (i *itemIterator) initBuffer() error {
	// defaults -- zero everything
	i.buffer = make([]*Item, 0)
	i.bufferI = 0
	i.lastFetchedCol = []byte{}
	i.startCol = []byte{}
	i.endOfRange = false
	i.currItem = nil
	i.currErr = nil
	i.lastId = ""

	// swap start/end if needed -- for reversed, the start time should be later
	if i.Reverse && i.start.Before(i.end) {
		i.start, i.end = i.end, i.start
	} else if !i.Reverse && i.start.After(i.end) {
		i.start, i.end = i.end, i.start
	}

	// setup the first time we fetch from - based on either the start time, or the "fromId" if we have one
	if i.fromId != "" {
		var err error
		_, t, err := columnNameToTimeId([]byte(i.fromId))
		if err != nil {
			return fmt.Errorf("Invalid `fromId` '%s' to paginate from: %v", i.fromId, err)
		}
		i.fetchTime = t.Truncate(i.ts.RowGranularity)
		i.startCol = []byte(i.fromId)
	} else {
		i.fetchTime = i.start.Truncate(i.ts.RowGranularity)
		if i.Reverse {
			i.startCol = []byte(fmt.Sprintf("%v§", i.start.Unix()))
		} else {
			i.startCol = []byte(fmt.Sprintf("%v", i.start.Unix()))
		}
	}

	return nil
}
