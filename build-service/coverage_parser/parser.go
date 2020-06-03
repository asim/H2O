package coverage_parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hailo-platform/H2O/build-service/models"
)

const (
	packagesFrom = "workspace"
)

var (
	regPercentage   = regexp.MustCompile(`[0-9]{1,3}\.[0-9]{1,2}%`)
	regPackagePath  = regexp.MustCompile(`_.*?\s`)
	regCoverageLine = regexp.MustCompile(`ok.*?coverage`)
)

func parseLine(line string) (models.Coverage, error) {
	coverage := models.Coverage{}

	s := regPercentage.FindString(line)
	perc, err := strconv.ParseFloat(s[:len(s)-1], 64)
	if err != nil {
		return coverage, fmt.Errorf("Couldn't parse percentage: %v", err)
	}
	coverage.Percentage = perc

	packagePath := regPackagePath.FindString(line)
	packagePath = strings.TrimSpace(packagePath)
	index := strings.Index(packagePath, packagesFrom)

	if index == -1 {
		return coverage, fmt.Errorf("Couldn't parse package name")
	}
	index += len(packagesFrom)

	coverage.PackageName = strings.TrimPrefix(packagePath[index:], "/")
	if coverage.PackageName == "" {
		coverage.PackageName = "main" // Packages at the root of a service are normally main
	}

	return coverage, nil
}

func getCoverage(from io.Reader) ([]models.Coverage, error) {
	coverage := make([]models.Coverage, 0)
	scanner := bufio.NewScanner(from)
	for scanner.Scan() {
		if !regCoverageLine.MatchString(scanner.Text()) {
			continue
		}
		c, err := parseLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		coverage = append(coverage, c)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return coverage, nil
}

func writeCoverage(w io.Writer, coverage []models.Coverage) error {
	data := make(map[string]float64) // Format the build service expects
	for _, c := range coverage {
		data[c.PackageName] = c.Percentage
	}

	enc := json.NewEncoder(w)
	return enc.Encode(data)
}

func CoverageMain() {
	coverage, err := getCoverage(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	if err := writeCoverage(os.Stdout, coverage); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	return
}
