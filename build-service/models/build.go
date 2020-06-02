package models

import (
	"time"
)

// Build stores metadata relating to a specific build
type Build struct {
	Hostname       string               `validate:"nonblank"` // The hostname that did the build
	Architecture   string               `validate:"nonblank"` // 386, AMD64 etc
	GoVersion      string               `validate:""`         // Version of Go used to build the binary
	SourceURL      string               `validate:"nonblank"` // The VCS url, down to the commit level
	BinaryURL      string               `validate:"nonblank"` // The location of the binary or JAR
	Version        string               `validate:"nonblank"` // Initially a human readable date. Eg. 20130601114431
	Language       string               `validate:"nonblank"` // Programming language
	Name           string               `validate:"nonblank"` // The service name
	Branch         string               `validate:"nonblank"` // The Git branch
	TimeStamp      int64                // UTC unix timestamp
	Coverage       map[string]float64   `json:"Coverage,omitempty"` // The code coverage as package => percentage
	Dependencies   map[string]string    `json:",omitempty"`         // The dependencies as importPath => commit
	MergeBaseDates map[string]time.Time `json:",omitempty"`         // The merge base dates of dependency commits
}

type CoverageSnapshot struct {
	Coverages []Coverage
	Branch    string
	Version   string
	Timestamp int64
}

type CoverageSnapshots []CoverageSnapshot

func (c CoverageSnapshots) Len() int           { return len(c) }
func (c CoverageSnapshots) Less(i, j int) bool { return c[i].Timestamp < c[j].Timestamp }
func (c CoverageSnapshots) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

type Coverage struct {
	PackageName string
	Percentage  float64
}
