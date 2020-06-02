// Offers up a recipe built on Gossie to make dealing with time indexes simple.
package timeseries

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/gossie/src/gossie"
)

/*

on mutate, add each "row key" to a single row (under 2ndary index row key) within the "index" CF
on read, we can load the index (whole row) and then test against it when scanning (if enabled)
for open-ended, we can load index and then work out start / end date from this (and do normal iterator)

*/

// Marshaler turns each item into a unique ID and time (that we index under)
type Marshaler func(i interface{}) (uid string, t time.Time)

// SecondaryIndexer turns each item into a secondary index ID
type SecondaryIndexer func(i interface{}) (index string)

// Ttler turns each item into a TTL, for ageing out data stored within a timeseries
type Ttler func(i interface{}) int32

// TimeSeries represents a single C* time series index on some data
type TimeSeries struct {
	Ks, Cf string
	// RowGranularity defines how small or large our row sizes are (how much time each represents)
	RowGranularity time.Duration
	// Marshaler is used to turn a an interface{} into a UID and time (for each item)
	Marshaler Marshaler
	// Secondary indexer is either nil (if none) or a function to extract the secondary
	// index ID from each item
	SecondaryIndexer SecondaryIndexer
	// IndexCf defines whether we should keep an overall index of the data (meaning we can skip rows on read and also do open-ended iterators)
	IndexCf string
	// Ttler is a function for calculating a TTL to apply to written columns
	Ttler Ttler
}

func (ts *TimeSeries) RowKeyAndColumnName(item interface{}) ([]byte, []byte) {
	uid, t := ts.Marshaler(item)
	rowPrefix := ""
	if ts.SecondaryIndexer != nil {
		rowPrefix = ts.SecondaryIndexer(item)
	}
	rowKey := ts.toRowKey(rowPrefix, t)
	columnName := idTimeToColumnName(uid, t)
	return rowKey, columnName
}

// Delete an item from the timeseries (remove the column entirely)
func (ts *TimeSeries) Delete(writer gossie.Writer, item interface{}) error {
	// While this doesn't currently return any error, adding it to the exposed interface so one can be returned later if
	// necessary (and clients will expect this)

	rowKey, columnName := ts.RowKeyAndColumnName(item)
	writer.DeleteColumns(ts.Cf, rowKey, [][]byte{columnName})
	return nil
}

// Map will write in any mutations needed to maintain this timeseries index
// based on the current item and the lastRead item
func (ts *TimeSeries) Map(writer gossie.Writer, item, lastRead interface{}) error {
	uid, t := ts.Marshaler(item)
	b, err := json.Marshal(item)
	if err != nil {
		log.Errorf("Error marshalling item %s", err) // TODO should probably bubble this up
		return err
	}
	rowPrefix := ""
	if ts.SecondaryIndexer != nil {
		rowPrefix = ts.SecondaryIndexer(item)
	}

	// C* ttl for ageing out columns
	var ttl int32
	if ts.Ttler != nil {
		ttl = ts.Ttler(item)
	}

	// always mutate against current marshaled value -- if not zero time
	if !t.IsZero() {
		tsRow := &gossie.Row{
			Key: ts.toRowKey(rowPrefix, t),
			Columns: []*gossie.Column{
				{
					Name:  idTimeToColumnName(uid, t),
					Value: b,
					Ttl:   ttl,
				},
			},
		}
		writer.Insert(ts.Cf, tsRow)
	}

	// old one that's different (and old one not zero)? add deletion...
	if lastRead != nil && !reflect.ValueOf(lastRead).IsNil() {
		lUid, lT := ts.Marshaler(lastRead)
		if ts.SecondaryIndexer != nil {
			lRowPrefix := ts.SecondaryIndexer(lastRead)
			if !lT.IsZero() && (!lT.Equal(t) || lRowPrefix != rowPrefix) {
				writer.DeleteColumns(ts.Cf, ts.toRowKey(lRowPrefix, lT), [][]byte{idTimeToColumnName(lUid, lT)})
			}
		}
	}

	// update index, if we have one defined
	if ts.IndexCf != "" {
		i := newIndex(ts, rowPrefix)
		i.write(writer, t.Truncate(ts.RowGranularity))
	}

	return nil
}

// toRowKey will turn an item plus the pre-marshaled time into a row key within C*
// This will take into account secondary index row-prefix, if defined
func (ts *TimeSeries) toRowKey(rowPrefix string, t time.Time) []byte {
	return []byte(fmt.Sprintf("%s%v", rowPrefix, t.Truncate(ts.RowGranularity).Unix()))
}

// Iterator yields a new iterator that will loop through all time series Items within
// the specified range - loading from C* on-demand
func (ts *TimeSeries) Iterator(start, end time.Time, from, secondaryIndex string) Iterator {
	return &itemIterator{
		ts:             ts,
		index:          newIndex(ts, secondaryIndex),
		start:          start,
		end:            end,
		fromId:         from,
		secondaryIndex: secondaryIndex,
		BatchSize:      50,
	}
}

// ReversedIterator yields a new iterator that will loop through all time series Items within
// the specified range - loading from C* on-demand - in reverse order
func (ts *TimeSeries) ReversedIterator(start, end time.Time, from, secondaryIndex string) Iterator {
	return &itemIterator{
		ts:             ts,
		index:          newIndex(ts, secondaryIndex),
		start:          start,
		end:            end,
		fromId:         from,
		secondaryIndex: secondaryIndex,
		BatchSize:      50,
		Reverse:        true,
	}
}

// UnboundedIterator yields a new iterator that will loop through every item within a time series
func (ts *TimeSeries) UnboundedIterator(secondaryIndex string) (Iterator, error) {
	if ts.IndexCf == "" {
		return nil, fmt.Errorf("Cannot use UnboundedIterator without an IndexCf being defined.")
	}
	i := newIndex(ts, secondaryIndex)
	start, end, err := i.bounds()
	return &itemIterator{
		ts:             ts,
		index:          newIndex(ts, secondaryIndex),
		start:          start,
		end:            end,
		fromId:         "",
		secondaryIndex: secondaryIndex,
		BatchSize:      50,
		Reverse:        false,
	}, err
}

func columnNameToTimeId(colName []byte) (string, time.Time, error) {
	parts := strings.Split(string(colName), "-")
	if len(parts) < 2 {
		return "", time.Now(), fmt.Errorf("Expecting column name of TS-ID (>= 2 parts) - got %v parts (%v)", len(parts), string(colName))
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "", time.Now(), fmt.Errorf("Expecting UNIX timestamp within TS-ID (%+v): %v", string(colName), err)
	}

	if ts > math.MaxInt32 {
		// Timestamp is not in seconds, assume nanos (i.e. came from a UnixNano() call)
		return strings.Join(parts[1:], "-"), time.Unix(0, ts), nil
	}
	return strings.Join(parts[1:], "-"), time.Unix(ts, 0), nil
}

func idTimeToColumnName(uid string, t time.Time) []byte {
	return []byte(fmt.Sprintf("%v-%s", t.Unix(), uid))
}

func itemToColumnName(item *Item) []byte {
	return idTimeToColumnName(item.id, item.t)
}

func colToItem(c *gossie.Column) (*Item, error) {
	id, t, err := columnNameToTimeId(c.Name)
	if err != nil {
		return nil, err
	}
	return &Item{
		id: id,
		t:  t,
		b:  c.Value,
	}, nil
}
