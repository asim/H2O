package util

import (
	"encoding/csv"
	log "github.com/cihub/seelog"
	"io"
	"os"
)

// Returns map of azName to rabbit hosts
func GetRabbitHosts() (map[string][]string, error) {

	file, err := os.Open("/etc/h2o/rabbithosts")
	if err != nil {
		log.Criticalf("Couldn't read file /etc/h2o/rabbithosts This file should contain the rabbit clusters: %v", err)
		return nil, err
	}
	defer file.Close()
	rdr := csv.NewReader(file)
	retMap := make(map[string][]string)
	for {
		line, err := rdr.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Criticalf("Couldn't read file /etc/h2o/rabbithosts This file should contain the rabbit clusters: %v", err)
			return nil, err
		}
		host := line[0]
		azName := line[1]
		list, ok := retMap[azName]
		if !ok {
			list = make([]string, 0)
		}
		list = append(list, host)
		retMap[azName] = list
	}
	return retMap, nil
}
