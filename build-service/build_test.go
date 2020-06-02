package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/HailoOSS/build-service/models"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func validBuild() models.Build {
	return models.Build{
		Hostname:     "localhost",
		Architecture: "amd64",
		GoVersion:    "1.1.1",
		SourceURL:    "https://github.com/HailoOSS/build-service/commit/53d6db9a88494e948b64415f53e1bf9da7efcc4b",
		BinaryURL:    "http://s3.amazon.com/abcdefg",
		Version:      "20130627091746",
		Language:     "Go",
		Name:         "com.HailoOSS.kernel.build-service",
		Branch:       "master",
		TimeStamp:    1372346773,
		Coverage: map[string]float64{
			"dao":    12.3,
			"domain": 100.0,
		},
		Dependencies: map[string]string{
			"github.com/HailoOSS/go-server-layer": "e6dc54ee3618c7b354dccdb6425cf4f82e07423c",
		},
	}
}

func TestValidateBuild(t *testing.T) {
	validBuild := validBuild()

	missingHostname := validBuild
	missingHostname.Hostname = ""

	testCases := []struct {
		build              *models.Build
		expectedErrorCount int
	}{
		{&validBuild, 0},
		{&missingHostname, 1},
	}

	for _, tc := range testCases {
		errors := ValidateBuild(tc.build)
		if len(errors) != tc.expectedErrorCount {
			t.Errorf("Expected %v errors, got %v", tc.expectedErrorCount, len(errors))
		}
	}
}

func TestGetNames(t *testing.T) {
	testCases := []struct {
		query          string
		expectedFilter string
	}{
		{"/builds/names", ""},
		{"/builds/names?filter=abc", "abc"},
	}

	for i, tc := range testCases {
		rw := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", tc.query, nil)

		repo := newTestRepo()
		buildRepo = repo
		commitRepo = newTestCommitRepo()

		getNamesHandler(rw, req)

		if repo.called != "GetNames" {
			t.Errorf("Didn't call GetNames (%v)", i)
		}
		if repo.filter != tc.expectedFilter {
			t.Errorf("Expected filter %v, got %v", tc.expectedFilter, repo.filter)
		}
	}
}

// Test that posted data is added to the repo intact
func TestCreateBuild(t *testing.T) {
	recorder := httptest.NewRecorder()

	sampleBuild := validBuild()

	data, _ := json.Marshal(sampleBuild)
	req, _ := http.NewRequest("POST", "/builds", bytes.NewReader(data))

	repo := newTestRepo()
	buildRepo = repo

	createBuildHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected %v, Got %v", http.StatusOK, recorder.Code)
	}

	if len(repo.builds) != 1 {
		t.Fatalf("Expected 1 build, got %v", len(repo.builds))
	}

	firstItem := repo.builds[0]

	if !reflect.DeepEqual(firstItem, &sampleBuild) {
		t.Errorf("\nExpected:%#v\nGot     :%#v", &sampleBuild, firstItem)
	}
}

func TestGetBuilds(t *testing.T) {
	testCases := []struct {
		repo           *memoryRepo
		reqPath        string
		expectedMethod string
	}{
		{newTestRepo(), "/builds", "GetAll"},
		{newTestRepo(), "/builds?:name=com.hailo.test", "GetAllWithName"},
		{newTestRepo(), "/builds?:name=com.hailo.test&:version=123", "GetVersion"},
	}

	for _, tc := range testCases {
		recorder := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "http://localhost"+tc.reqPath, nil)

		buildRepo = tc.repo
		getBuildsHandler(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("Expected %v, Got %v", http.StatusOK, recorder.Code)
		}

		if tc.expectedMethod != tc.repo.called {
			t.Errorf("Expected %v to have been called. Got: %v", tc.expectedMethod, tc.repo.called)
		}
	}
}

func TestGetCoverage(t *testing.T) {
	recorder := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/builds?:name=&:version=123", nil)

	repo := newTestRepo()
	buildRepo = repo

	getCoverageHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected %v, Got %v", http.StatusOK, recorder.Code)
	}

	if repo.called != "GetCoverage" {
		t.Errorf("Expected GetCoverage to have been called. Got: %v", repo.called)
	}
}

func TestGetCoverageTrend(t *testing.T) {
	recorder := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/builds/?:name=&", nil)

	repo := newTestRepo()
	buildRepo = repo

	getCoverageTrendHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected %v, Got %v", http.StatusOK, recorder.Code)
	}

	if repo.called != "GetCoverageTrend" {
		t.Errorf("Expected GetCoverage to have been called. Got: %v", repo.called)
	}
}

func TestGroupCoverageRows(t *testing.T) {
	coverageRows := []coverageRow{
		{
			service:    "service",
			version:    "123",
			branch:     "master",
			pkg:        "pkg1",
			percentage: 50.00,
			timestamp:  312234234,
		},
		{
			service:    "service",
			version:    "123",
			branch:     "master",
			pkg:        "pkg2",
			percentage: 60.00,
			timestamp:  312234234,
		},
		{
			service:    "service",
			version:    "124",
			branch:     "master",
			pkg:        "pkg1",
			percentage: 51.00,
			timestamp:  312234235,
		},
		{
			service:    "service",
			version:    "124",
			branch:     "master",
			pkg:        "pkg2",
			percentage: 61.00,
			timestamp:  312234235,
		},
	}
	expected := models.CoverageSnapshots{
		{
			Timestamp: 312234234,
			Branch:    "master",
			Version:   "123",
			Coverages: []models.Coverage{
				models.Coverage{
					PackageName: "pkg1",
					Percentage:  50.00,
				},
				models.Coverage{
					PackageName: "pkg2",
					Percentage:  60.00,
				},
			},
		},
		{
			Timestamp: 312234235,
			Branch:    "master",
			Version:   "124",
			Coverages: []models.Coverage{
				models.Coverage{
					PackageName: "pkg1",
					Percentage:  51.00,
				},
				models.Coverage{
					PackageName: "pkg2",
					Percentage:  61.00,
				},
			},
		},
	}

	actual := groupCoverageRows(coverageRows)
	if !reflect.DeepEqual(actual, expected) {
		t.Error("Expected:")
		t.Errorf("%+v", expected)
		t.Error("Actual:")
		t.Errorf("%+v", actual)
	}
}

func TestDeleteBuild(t *testing.T) {
	testCases := []struct {
		path           string
		repo           *memoryRepo
		expectedMethod string
		expectedStatus int
	}{
		{"/builds?:name=com.test&:version=123", newTestRepo(), "Delete", http.StatusOK},
		{"/builds?:name=&:version=123", newTestRepo(), "", http.StatusBadRequest},
		{"/builds?:name=com.test&:version=", newTestRepo(), "", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		recorder := httptest.NewRecorder()

		req, _ := http.NewRequest("DELETE", tc.path, nil)

		buildRepo = tc.repo

		deleteBuildHandler(recorder, req)

		if recorder.Code != tc.expectedStatus {
			t.Errorf("Expectec status %v, got %v", tc.expectedStatus, recorder.Code)
			continue
		}

		if tc.expectedMethod != tc.repo.called {
			t.Errorf("Expected %v to have been called. Got: %v", tc.expectedMethod, tc.repo.called)
		}
	}
}

func TestMissingServiceName(t *testing.T) {
	recorder := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/builds?:name=&:version=123", nil)

	repo := newTestRepo()
	buildRepo = repo

	getBuildsHandler(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected bad request, got %v", recorder.Code)
	}
}
