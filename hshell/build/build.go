package build

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HailoOSS/build-service/models"
	mhttp "github.com/mreiferson/go-httpclient"
)

// Use one client for all requests
var client *http.Client

var BuildUrl string = "https://build-api.elasticride.com"
var apiKey string = os.Getenv("BUILD_SERVICE_API_KEY")

func init() {
	transport := &mhttp.Transport{
		ConnectTimeout:        1 * time.Second,
		RequestTimeout:        5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}

	client = &http.Client{Transport: transport}
}

func GetAllBuilt() []string {
	builds, _ := callBuildNames("")

	return builds
}

func GetAllVersions(service string) []string {
	builds, err := callBuilds(service)
	if err != nil {
		return make([]string, 0)
	}
	buildstr := make([]string, len(builds))
	for i, bld := range builds {
		buildstr[i] = bld.Version
	}
	return buildstr
}

var bufferedBranch map[string]string = make(map[string]string)
var bufferedGitHash map[string]string = make(map[string]string)

func GetBranchBuffered(service string, version string) (string, string) {
	if bufferedBranch[service+version] == "" {
		GetBranch(service, version)
	} else {
		go GetBranch(service, version)
	}

	return bufferedBranch[service+version], bufferedGitHash[service+version]
}

func GetBranch(service string, version string) (string, string) {
	bld, err := callBuild(service, version)
	if err != nil {
		fmt.Println("Call to build service failed.  Is your VPN connected to Global01-test?")
		return "", ""
	}

	if bld.Version == version {
		bufferedBranch[service+version] = bld.Branch
		bufferedGitHash[service+version] = strings.TrimPrefix(strings.SplitAfterN(bld.SourceURL, "commit", 2)[1], "/")[:7]
		return bufferedBranch[service+version], bufferedGitHash[service+version]
	}

	return "", ""
}

func GetLatestVersion(service string) string {
	buildArr := GetAllVersions(service)
	if len(buildArr) == 0 {
		return ""
	}
	sort.Strings(buildArr)
	return buildArr[len(buildArr)-1]
}

func GetLatestVersionBranch(service, branch string) int {
	builds, err := callBuilds(service)
	if err != nil {
		fmt.Printf("Unable to get latest branch version: %v \n", err)
		return 0
	}

	latestVersion := 0
	for _, build := range builds {
		if build.Branch == branch {
			v, _ := strconv.Atoi(build.Version)
			if v > latestVersion {
				latestVersion = v
			}
		}
	}

	return latestVersion
}

func callBuild(service string, version string) (*models.Build, error) {

	build := &models.Build{}

	req, _ := http.NewRequest("GET", BuildUrl+"/"+service+"/"+version, nil)
	if apiKey != "" {
		req.Header.Add("x-api-key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return build, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(build)
	if err != nil {
		return build, err
	}

	return build, nil
}

func callBuilds(service string) ([]models.Build, error) {
	builds := make([]models.Build, 0)

	req, _ := http.NewRequest("GET", BuildUrl+"/"+service, nil)
	if apiKey != "" {
		req.Header.Add("x-api-key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return builds, err
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&builds)
	if err != nil {
		return builds, err
	}

	return builds, nil
}

func callBuildNames(filter string) ([]string, error) {

	builds := make([]string, 0)
	if len(filter) != 0 {
		filter = fmt.Sprintf("?filter=%s", filter)
	}

	req, _ := http.NewRequest("GET", BuildUrl+"/names"+filter, nil)
	if apiKey != "" {
		req.Header.Add("x-api-key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return builds, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&builds)
	if err != nil {
		return builds, err
	}

	return builds, nil
}
