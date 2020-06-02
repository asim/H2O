package dao

import (
	"github.com/HailoOSS/gossie/src/gossie"
)

type rowProvider struct {
	mapping     gossie.Mapping
	buffer      []*gossie.Row
	row         *gossie.Row
	position    int
	columnLimit int
}

func (r *rowProvider) feedRow() error {
	if r.row == nil {
		if len(r.buffer) <= 0 {
			return gossie.Done
		}
		r.row = r.buffer[0]
		r.position = 0
		r.buffer = r.buffer[1:len(r.buffer)]
	}
	return nil
}

func (r *rowProvider) Key() ([]byte, error) {
	if err := r.feedRow(); err != nil {
		return nil, err
	}
	return r.row.Key, nil
}

func (r *rowProvider) NextColumn() (*gossie.Column, error) {
	if err := r.feedRow(); err != nil {
		return nil, err
	}
	if r.position >= len(r.row.Columns) {
		if r.position >= r.columnLimit {
			return nil, gossie.EndAtLimit
		} else {
			return nil, gossie.EndBeforeLimit
		}
	}
	c := r.row.Columns[r.position]
	r.position++
	return c, nil
}

func (r *rowProvider) Rewind() {
	r.position--
	if r.position < 0 {
		r.position = 0
	}
}

func (r *rowProvider) Next(destination interface{}) error {
	err := r.mapping.Unmap(destination, r)
	if err == gossie.Done {
		// force new row feed and try again, just once
		r.row = nil
		err = r.mapping.Unmap(destination, r)
	}
	return err
}
