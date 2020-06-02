package cassandra

import (
	"github.com/HailoOSS/gossie/src/gossie"
)

type MultiRowProvider struct {
	// mapping for unmarshaling a row into a struct
	Mapping gossie.Mapping
	// our buffer of rows
	Buffer []*gossie.Row
	// how many columns to limit to
	ColumnLimit int

	row      *gossie.Row
	position int
}

func (r *MultiRowProvider) feedRow() error {
	if r.row == nil {
		if len(r.Buffer) <= 0 {
			return gossie.Done
		}
		r.row = r.Buffer[0]
		r.position = 0
		r.Buffer = r.Buffer[1:len(r.Buffer)]
	}
	return nil
}

func (r *MultiRowProvider) Key() ([]byte, error) {
	if err := r.feedRow(); err != nil {
		return nil, err
	}
	return r.row.Key, nil
}

func (r *MultiRowProvider) NextColumn() (*gossie.Column, error) {
	if err := r.feedRow(); err != nil {
		return nil, err
	}
	if r.position >= len(r.row.Columns) {
		if r.position >= r.ColumnLimit {
			return nil, gossie.EndAtLimit
		} else {
			return nil, gossie.EndBeforeLimit
		}
	}
	c := r.row.Columns[r.position]
	r.position++
	return c, nil
}

func (r *MultiRowProvider) Rewind() {
	r.position--
	if r.position < 0 {
		r.position = 0
	}
}

func (r *MultiRowProvider) Next(destination interface{}) error {
	err := r.Mapping.Unmap(destination, r)
	if err == gossie.Done {
		// force new row feed and try again, just once
		r.row = nil
		err = r.Mapping.Unmap(destination, r)
	}
	return err
}
