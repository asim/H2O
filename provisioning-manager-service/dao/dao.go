package dao

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/cassandra"
	"github.com/HailoOSS/gossie/src/gossie"
)

const (
	keyspace     = "provisioning"
	columnFamily = "provisioned_service"
)

var (
	mapping    gossie.Mapping
	rlmapping  gossie.Mapping
	rlsmapping gossie.Mapping
)

func init() {
	var err error
	log.Info("Initialising Gossie mapping")
	mapping, err = gossie.NewMapping(&Service{})
	if err != nil {
		log.Flush()
		panic(fmt.Sprintf("Invalid mapping - unexpected error: %v", err))
	}
	rlmapping, err = gossie.NewMapping(&storedRunLevel{})
	if err != nil {
		log.Flush()
		panic(fmt.Sprintf("Invalid mapping - unexpected error: %v", err))
	}
	rlsmapping, err = gossie.NewMapping(&storedServiceRunLevels{})
	if err != nil {
		log.Flush()
		panic(fmt.Sprintf("Invalid mapping - unexpected error: %v", err))
	}
}

func checkDateNum(dateNum uint64) error {
	year := uint64(dateNum / 10000000000)
	if year < 2013 {
		return errors.New("Year part of date was out of range (< 2013)")
	}

	dateNum -= (year * 10000000000)
	month := uint64(dateNum / 100000000)
	if month < 1 || month > 12 {
		return errors.New("Month part of date was out of range")
	}

	dateNum -= (month * 100000000)
	day := uint64(dateNum / 1000000)
	if day < 1 || day > 31 {
		// TODO: check day number is valid for given month
		return errors.New("Day part of date was out of range")
	}

	dateNum -= (day * 1000000)
	hours := uint64(dateNum / 10000)
	if hours >= 24 {
		return errors.New("Hours part of date was out of range")
	}

	dateNum -= (hours * 10000)
	minutes := uint64(dateNum / 100)
	if minutes >= 60 {
		return errors.New("Minutes part of date was out of range")
	}

	seconds := dateNum - (minutes * 100)
	if seconds >= 60 {
		return errors.New("Seconds part of date was out of range")
	}

	return nil
}

func generateHash(vals ...string) string {
	joined := strings.Join(vals, "\t")
	hasher := sha1.New()
	hasher.Write([]byte(joined))
	return hex.EncodeToString(hasher.Sum(nil))
}

func Create(s *Service) error {
	if len(s.ServiceName) == 0 {
		return errors.New("Empty service name")
	}

	if err := checkDateNum(s.ServiceVersion); err != nil {
		return err
	}

	if len(s.MachineClass) == 0 {
		return errors.New("Empty machine class")
	}

	if s.NoFileSoftLimit < 1024 {
		s.NoFileSoftLimit = 1024
	}

	if s.NoFileHardLimit < 1024 {
		s.NoFileHardLimit = 4096
	}

	s.Id = s.id()

	row, err := mapping.Map(s)
	if err != nil {
		return fmt.Errorf("Error mapping provisioned service: %v", err)
	}
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}
	if err := pool.Writer().Insert(columnFamily, row).Run(); err != nil {
		return fmt.Errorf("Create error writing to C*: %v", err)
	}

	return nil
}

func Delete(s *Service) error {
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return fmt.Errorf("Failed to get connection pool: %v", err)
	}
	if err := pool.Writer().Delete(columnFamily, []byte(s.id())).Run(); err != nil {
		return fmt.Errorf("C* delete failed: %v", err)
	}

	return nil
}

func Provisioned(machineClass string) ([]*Service, error) {
	var getter int
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}
	reader := pool.Reader().Cf(columnFamily)
	if len(machineClass) > 0 {
		getter = 1
		reader = reader.Where([]byte("machineclass"), gossie.EQ, []byte(machineClass))
	}

	var provisioned []*Service
	var rows []*gossie.Row
	start := []byte{}

	for {
		switch getter {
		case 0:
			rows, err = reader.RangeGet(&gossie.Range{Start: start, Count: 100})
		case 1:
			rows, err = reader.IndexedGet(&gossie.IndexedRange{Start: start, Count: 100})
		}

		if err != nil {
			return nil, fmt.Errorf("C* get query failed: %v", err)
		}

		if len(start) > 0 {
			// If we've run out of results then we're done.
			if len(rows) < 2 {
				break
			}
			// Otherwise strip the starting row.
			rows = rows[1:]
		}

		result := &rowProvider{
			columnLimit: 100,
			buffer:      rows,
			mapping:     mapping,
		}

		for {
			st := &Service{}
			err := result.Next(st)
			if err != nil { // Done is also returned in err
				break
			}
			provisioned = append(provisioned, st)
		}

		if len(rows) > 0 {
			start = rows[len(rows)-1].Key
		} else {
			break
		}
	}

	return provisioned, nil
}

func Read(name string, version uint64, class string) (*Service, error) {
	pool, err := cassandra.ConnectionPool(keyspace)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection pool: %v", err)
	}

	query := pool.Query(mapping)
	result, err := query.Get(generateHash(name, strconv.FormatUint(version, 10), class))
	if err != nil {
		return nil, fmt.Errorf("C* query failed: %v", err)
	}

	res := &Service{}
	if err := result.Next(res); err != nil {
		return nil, fmt.Errorf("Failed to read result %v", err)
	}

	return res, nil
}
