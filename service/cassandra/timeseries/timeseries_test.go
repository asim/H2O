package timeseries_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/service/cassandra/timeseries"
)

func TestDeleteWithoutSecondaryIndex(t *testing.T) {
	cassandra.PoolConstructor = cassandra.NewMockConnectionPool
	defer func() {
		cassandra.PoolConstructor = cassandra.DefaultPoolConstructor
	}()

	testDate := time.Date(2014, 1, 1, 0, 30, 0, 0, time.UTC)
	// Hourly granularity sticks this in at midnight
	testBucketedDate := time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)

	testTs := &timeseries.TimeSeries{
		Ks:             "testTsKs",
		Cf:             "testTsCf",
		RowGranularity: time.Hour,
		Marshaler: func(i interface{}) (uid string, t time.Time) {
			return "testUid", testDate
		},
		IndexCf: "testTsIndexCf",
	}

	pool, err := cassandra.ConnectionPool(testTs.Ks)
	assert.Nil(t, err, "Error getting C* connection pool")

	writer := pool.Writer().(*cassandra.MockWriter)
	columnNames := [][]byte{
		[]byte(fmt.Sprintf("%d-testUid", testDate.Unix())),
	}
	rowName := []byte(fmt.Sprintf("%d", testBucketedDate.Unix()))
	writer.On("DeleteColumns", testTs.Cf, rowName, columnNames).Return(nil).Once()
	testTs.Delete(writer, true)
}

func TestDeleteWithSecondaryIndex(t *testing.T) {
	cassandra.PoolConstructor = cassandra.NewMockConnectionPool
	defer func() {
		cassandra.PoolConstructor = cassandra.DefaultPoolConstructor
	}()

	testDate := time.Date(2014, 1, 1, 0, 30, 0, 0, time.UTC)
	// Hourly granularity sticks this in at midnight
	testBucketedDate := time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)

	testTs := &timeseries.TimeSeries{
		Ks:             "testTsKs",
		Cf:             "testTsCf",
		RowGranularity: time.Hour,
		Marshaler: func(i interface{}) (string, time.Time) {
			return "testUid", testDate
		},
		IndexCf: "testTsIndexCf",
		SecondaryIndexer: func(i interface{}) string {
			return "testTsSecondaryIndex"
		},
	}

	pool, err := cassandra.ConnectionPool(testTs.Ks)
	assert.Nil(t, err, "Error getting C* connection pool")

	writer := pool.Writer().(*cassandra.MockWriter)
	columnNames := [][]byte{
		[]byte(fmt.Sprintf("%d-testUid", testDate.Unix())),
	}
	rowName := []byte(fmt.Sprintf("testTsSecondaryIndex%d", testBucketedDate.Unix()))
	writer.On("DeleteColumns", testTs.Cf, rowName, columnNames).Return(nil).Once()
	testTs.Delete(writer, true)
}
