package cassandra

import (
	"github.com/HailoOSS/gossie/src/gossie"
	"github.com/stretchr/testify/mock"
)

type MockConnectionPool struct {
	mock.Mock

	reader *MockReader
	writer *MockWriter
	query  *MockQuery
}

type MockReader struct {
	mock.Mock
}

type MockWriter struct {
	mock.Mock
}

type MockQuery struct {
	mock.Mock
	gossie.Query
}

func NewMockConnectionPool(nodes []string, keyspace string, options gossie.PoolOptions) (gossie.ConnectionPool, error) {
	impl := &MockConnectionPool{
		writer: &MockWriter{},
		reader: &MockReader{},
	}
	return impl, nil
}

func (cp *MockConnectionPool) Writer() gossie.Writer {
	return cp.writer
}

func (cp *MockConnectionPool) Batch() gossie.Batch {
	return nil
}

func (cp *MockConnectionPool) Close() error {
	return nil
}

func (cp *MockConnectionPool) Keyspace() string {
	return "test"
}

func (cp *MockConnectionPool) Query(gossie.Mapping) gossie.Query {
	impl := &MockQuery{}
	return impl
}

func (cp *MockConnectionPool) Reader() gossie.Reader {
	return cp.reader
}

func (cp *MockConnectionPool) Schema() *gossie.Schema {
	return nil
}

// MockWriter methods

func (w *MockWriter) ConsistencyLevel(level int) gossie.Writer {
	return w
}

func (w *MockWriter) Insert(cf string, row *gossie.Row) gossie.Writer {
	// Deliberately do not pass along row; the test functions almost certainly won't have this
	w.Mock.Called(cf, nil)
	return w
}

func (w *MockWriter) InsertTtl(cf string, row *gossie.Row, ttl int) gossie.Writer {
	// Deliberately do not pass along row; the test functions almost certainly won't have this
	w.Mock.Called(cf, nil, ttl)
	return w
}

func (w *MockWriter) DeltaCounters(cf string, row *gossie.Row) gossie.Writer {
	return w
}

func (w *MockWriter) Delete(cf string, key []byte) gossie.Writer {
	// Deliberately do not pass along key; the test functions almost certainly won't have this
	w.Mock.Called(cf, nil)
	return w
}

func (w *MockWriter) DeleteColumns(cf string, key []byte, columns [][]byte) gossie.Writer {
	w.Mock.Called(cf, key, columns)
	return w
}

func (w *MockWriter) Run() error {
	w.Mock.Called()
	return nil
}

// MockReader methods

func (r *MockReader) ConsistencyLevel(level int) gossie.Reader {
	r.Mock.Called(level)

	return r
}
func (r *MockReader) Cf(cf string) gossie.Reader {
	r.Mock.Called(cf)

	return r
}
func (r *MockReader) Slice(slice *gossie.Slice) gossie.Reader {
	r.Mock.Called(slice)

	return r
}
func (r *MockReader) Columns(columns [][]byte) gossie.Reader {
	r.Mock.Called(columns)

	return r
}
func (r *MockReader) Where(column []byte, op gossie.Operator, value []byte) gossie.Reader {
	r.Mock.Called(column, op, value)

	return r
}
func (r *MockReader) Get(key []byte) (*gossie.Row, error) {
	ret := r.Mock.Called(key)

	var r0 *gossie.Row
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*gossie.Row)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (r *MockReader) MultiGet(keys [][]byte) ([]*gossie.Row, error) {
	ret := r.Mock.Called(keys)

	r0 := ret.Get(0).([]*gossie.Row)
	r1 := ret.Error(1)

	return r0, r1
}
func (r *MockReader) Count(key []byte) (int, error) {
	ret := r.Mock.Called(key)

	r0 := ret.Get(0).(int)
	r1 := ret.Error(1)

	return r0, r1
}
func (r *MockReader) MultiCount(keys [][]byte) ([]*gossie.RowColumnCount, error) {
	ret := r.Mock.Called(keys)

	r0 := ret.Get(0).([]*gossie.RowColumnCount)
	r1 := ret.Error(1)

	return r0, r1
}

func (r *MockReader) RangeGet(rang *gossie.Range) ([]*gossie.Row, error) {
	ret := r.Mock.Called(rang)

	r0 := ret.Get(0).([]*gossie.Row)
	r1 := ret.Error(1)

	return r0, r1
}
func (r *MockReader) IndexedGet(rang *gossie.IndexedRange) ([]*gossie.Row, error) {
	ret := r.Mock.Called(rang)

	r0 := ret.Get(0).([]*gossie.Row)
	r1 := ret.Error(1)

	return r0, r1
}
