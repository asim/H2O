//quick and dirty test script for the broker prototypes
//This measures client side CPU and request response timings.
package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"flag"
	"github.com/grd/stat"
	"log"
	"os"
	"sort"
	"strconv"
)

var jj int64 = 0
var origTime int64
var lastTime int64
var errorCount int64 = 0

func main() {

	inFile := flag.String("file", "", "The input file")
	flag.Parse()
	if *inFile == "" {
		flag.PrintDefaults()
		return
	}

	dat, err := os.Open(*inFile)
	if err != nil {
		log.Printf("error opening file: %v\n", err)
		return
	}
	rdr := bufio.NewReader(dat)
	resultData := stat.Float64Slice{}
	line, e := Readln(rdr)

	for e == nil {
		parseLogLine(line, &resultData)

		line, e = Readln(rdr)
	}

	getRoundStats(resultData)
}

// Readln returns a single line (without the ending \n)
// from the input buffered reader.
// An error is returned iff there is an error with the
// buffered reader.
func Readln(r *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

func getRoundStats(resultData stat.Float64Slice) {

	sort.Sort(resultData)

	log.Println("Timing:")
	log.Printf("\tcount = %v\n", resultData.Len())
	log.Printf("\tthroughput = %v\n", float64(resultData.Len())/(float64(lastTime-origTime)*1.0e-9))
	min, _ := stat.Min(resultData)
	log.Printf("\tmin = %v\n", min)
	max, _ := stat.Max(resultData)
	log.Printf("\tmax = %v\n", max)
	log.Printf("\tmean = %v\n", stat.Mean(resultData))
	log.Printf("\tstandard deviation = %v\n", stat.Absdev(resultData))
	log.Printf("\tUpper 90  = %v\n", stat.QuantileFromSortedData(resultData, 0.90))
	log.Printf("\tUpper 95  = %v\n", stat.QuantileFromSortedData(resultData, 0.95))
	log.Printf("\tUpper 99  = %v\n", stat.QuantileFromSortedData(resultData, 0.99))
	log.Printf("\tUpper 99.9  = %v\n", stat.QuantileFromSortedData(resultData, 0.999))
	log.Printf("\tUpper 99.99  = %v\n", stat.QuantileFromSortedData(resultData, 0.9999))
	log.Printf("\tNum Errors = %v\n", errorCount)

}

func parseLogLine(line string, roundResults *stat.Float64Slice) {

	//parses csv
	b := bytes.NewBufferString(line)
	csvReader := csv.NewReader(b)
	record, err := csvReader.Read()
	if err != nil {
		log.Println("error parsing csv", line)
		return
	}
	if record[3] == "false" {
		errorCount++
	}
	i, _ := strconv.Atoi(record[2])
	dur := int64(i)
	*roundResults = append(*roundResults, float64(dur)*1.0e-6)

	ts, _ := strconv.ParseInt(record[0], 10, 64)

	if jj == 0 {
		origTime = ts
	}
	lastTime = ts
	jj++
}
