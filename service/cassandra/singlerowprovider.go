package cassandra

import (
	"github.com/HailoOSS/gossie/src/gossie"
)

type SingleRowProvider struct {
	Row      *gossie.Row
	position int
}

func (r *SingleRowProvider) Key() ([]byte, error) {
	return r.Row.Key, nil
}

func (r *SingleRowProvider) NextColumn() (*gossie.Column, error) {
	if r.position >= len(r.Row.Columns) {
		return nil, gossie.EndAtLimit
	}
	c := r.Row.Columns[r.position]
	r.position++
	return c, nil
}

func (r *SingleRowProvider) Rewind() {
	r.position--
	if r.position < 0 {
		r.position = 0
	}
}
