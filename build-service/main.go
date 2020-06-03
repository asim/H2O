package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/pat"

	"github.com/hailo-platform/H2O/build-service/coverage_parser"
	"github.com/hailo-platform/H2O/build-service/models"
	"github.com/hailo-platform/H2O/build-service/validate"
)

const (
	langGo      = "Go"
	langJava    = "Java"
	serviceName = "com.hailo-platform/H2O.build-service"

	envSqlUsername = "BUILD_SERVICE_SQL_USERNAME"
	envSqlPassword = "BUILD_SERVICE_SQL_PASSWORD"
	envSqlServer   = "BUILD_SERVICE_SQL_SERVER"
	envSqlPort     = "BUILD_SERVICE_SQL_PORT"
	envSqlDatabase = "BUILD_SERVICE_SQL_DATABASE"
	envGithubToken = "BUILD_SERVICE_GITHUB_TOKEN"

	defaultLimit                 = 10
	defaultCoverageTrendDuration = -90 * 24 * time.Hour // 90 days

	defaultReadTimeout  = 30 * time.Second
	defaultWriteTimeout = 30 * time.Second
	defaultPort         = 3000
	defaultTlsAddr      = ":8443"
	defaultKey          = ""
	defaultCert         = ""
)

var (
	buildRepo     BuildRepository
	commitRepo    CommitRepo
	createTables  bool
	listenPort    int
	outputName    bool
	outputVersion bool
	runCoverage   bool
	tlsListAddr   string
)

// BuildRepository defines the interface required by a build data store
type BuildRepository interface {
	Create(b *models.Build) error
	GetAll(limit int) ([]*models.Build, error)
	GetAllWithName(name string, limit int) ([]*models.Build, error)
	GetVersion(name, version string) (*models.Build, error)
	Delete(name, version string) error
	GetNames(filter string) ([]string, error)
	GetCoverage(name, version string) (map[string]float64, error)
	GetCoverageTrend(name string, since time.Time) (models.CoverageSnapshots, error)
	SetMergeBaseDate(name, version, importPath, commit string, date time.Time) error
}

type CommitRepo interface {
	MergeBaseDate(importPath, sha, base string) (*time.Time, error)
}

func logHTTPError(rw http.ResponseWriter, err string, status int) {
	http.Error(rw, err, status)
	log.Println(err)
}

func getNamesHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("GET", r.URL)

	filter := r.URL.Query().Get("filter")
	names, err := buildRepo.GetNames(filter)
	if err != nil {
		log.Println(err)
		logHTTPError(rw, "Error getting names", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(names)
}

func createBuildHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("POST", r.URL)

	if r.Body == nil {
		logHTTPError(rw, "No POST body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	build := new(models.Build)

	err := json.NewDecoder(r.Body).Decode(build)
	if err != nil {
		logHTTPError(rw, "Error decoding JSON", http.StatusBadRequest)
		return
	}

	errors := validate.Validate(build)
	if len(errors) > 0 {
		logHTTPError(rw, "Invalid build", http.StatusBadRequest)
		return
	}

	err = buildRepo.Create(build)
	if err != nil {
		logHTTPError(rw, fmt.Sprintf("Error saving build: %v", err), http.StatusInternalServerError)
		return
	}

	go func() {
		for importPath, commit := range build.Dependencies {
			mergeBaseDate, err := commitRepo.MergeBaseDate(importPath, commit, "HEAD")
			if err != nil || mergeBaseDate == nil {
				log.Printf("Failed to get merge base date of %s/%s: %s", importPath, commit, err)
				continue
			}

			err = buildRepo.SetMergeBaseDate(build.Name, build.Version, importPath, commit, *mergeBaseDate)
			if err != nil {
				log.Printf("Failed to set commit date (%s, %s, %s, %s): %s", build.Name, build.Version, importPath, commit)
			}
		}
	}()
}

func deleteBuildHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("DELETE", r.URL)

	serviceName := r.URL.Query().Get(":name")
	buildVersion := r.URL.Query().Get(":version")

	if serviceName == "" {
		logHTTPError(rw, "Missing Service Name", http.StatusBadRequest)
		return
	}

	if buildVersion == "" {
		logHTTPError(rw, "Missing version", http.StatusBadRequest)
		return
	}

	err := buildRepo.Delete(serviceName, buildVersion)
	if err != nil {
		logHTTPError(rw, "Error deleting build", http.StatusInternalServerError)
		return
	}
}

func getBuildsHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("GET BUILDS", r.URL)

	serviceName := r.URL.Query().Get(":name")
	buildVersion := r.URL.Query().Get(":version")

	limitQuery := r.URL.Query().Get("limit")
	limit := defaultLimit
	if l, err := strconv.Atoi(limitQuery); err == nil {
		limit = l
	}

	// We're getting a list of builds
	if buildVersion == "" {
		var builds []*models.Build
		var err error

		if serviceName == "" {
			builds, err = buildRepo.GetAll(limit)
		} else {
			builds, err = buildRepo.GetAllWithName(serviceName, limit)
		}

		if err != nil {
			logHTTPError(rw, fmt.Sprintf("Error getting builds: %v", err), http.StatusInternalServerError)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(builds)
		return
	}

	// We're just getting one build
	if serviceName == "" {
		// Shouldn't have version and no name
		logHTTPError(rw, "No service name supplied", http.StatusBadRequest)
		return
	}

	build, err := buildRepo.GetVersion(serviceName, buildVersion)
	if err != nil {
		logHTTPError(rw, fmt.Sprintf("Error getting build: %v", err), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(build)
}

func getCoverageHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("GET COVERAGE", r.URL)

	serviceName := r.URL.Query().Get(":name")
	buildVersion := r.URL.Query().Get(":version")

	coverage, err := buildRepo.GetCoverage(serviceName, buildVersion)
	if err != nil {
		logHTTPError(rw, fmt.Sprintf("Error getting code coverage: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Coverage: %+v", coverage)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(coverage)
}

func getCoverageTrendHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("GET COVERAGE_TREND", r.URL)

	serviceName := r.URL.Query().Get(":name")
	since, err := time.Parse("20060102150405", r.URL.Query().Get("since"))
	if err != nil {
		since = time.Now().Add(defaultCoverageTrendDuration) // default to 90 days ago
	}

	trend, err := buildRepo.GetCoverageTrend(serviceName, since)
	if err != nil {
		logHTTPError(rw, fmt.Sprintf("Error getting code coverage trend: %v", err), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(trend)
}

// ValidateBuild checks that a build is valid
func ValidateBuild(build *models.Build) []error {
	return validate.Validate(build)
}

type allowRemoteHandler struct {
	h http.Handler
}

func (arh allowRemoteHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	arh.h.ServeHTTP(w, req)
}

func router() http.Handler {
	r := pat.New()

	r.Post("/builds", createBuildHandler)

	r.Delete("/builds/{name}/{version}", deleteBuildHandler)

	r.Get("/builds/names", getNamesHandler)
	r.Get("/builds/{name}/{version}/coverage", getCoverageHandler)
	r.Get("/builds/{name}/coverage", getCoverageTrendHandler)
	r.Get("/builds/{name}/{version}", getBuildsHandler)
	r.Get("/builds/{name}", getBuildsHandler)
	r.Get("/builds", getBuildsHandler)

	return allowRemoteHandler{r}
}

func checkEnv() bool {
	ok := true
	for _, s := range []string{envSqlServer, envSqlPort, envSqlUsername, envSqlDatabase} {
		val := os.Getenv(s)
		if val == "" {
			log.Println("Missing environment variable", s)
			ok = false
		}
	}

	return ok
}

func init() {
	flag.BoolVar(&createTables, "createtables", false, "Create the required DB tables")
	flag.BoolVar(&runCoverage, "coverage", false, "Run coverage and exit.")
	flag.IntVar(&listenPort, "port", defaultPort, "The listening port to bind HTTP to (default "+strconv.Itoa(defaultPort)+")")
	flag.BoolVar(&outputName, "name", false, "Print service name and exit.")
	flag.StringVar(&tlsListAddr, "tls", defaultTlsAddr, "The listening address to bind TLS to (default "+defaultTlsAddr+")")
	flag.BoolVar(&outputVersion, "version", false, "Print version and exit.")

}

func main() {
	flag.Parse()

	if outputName {
		fmt.Println(serviceName)
		return
	}

	if outputVersion {
		fmt.Println(ServiceVersion)
		return
	}

	if runCoverage {
		coverage_parser.CoverageMain()
		return
	}

	if !checkEnv() {
		return
	}

	repo := new(sqlRepo)
	err := repo.Connect(os.Getenv(envSqlServer), os.Getenv(envSqlPort), os.Getenv(envSqlUsername), os.Getenv(envSqlPassword), os.Getenv(envSqlDatabase))
	if err != nil {
		log.Println(err)
		return
	}

	if createTables {
		log.Println("Creating tables")
		err := repo.CreateTables()
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("OK")
		return
	}

	buildRepo = repo
	commitRepo = NewGithubRepo(os.Getenv(envGithubToken))

	r := router()
	s := http.Server{
		Addr:         tlsListAddr,
		Handler:      r,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
	}

	c := bytes.NewBufferString(defaultCert)
	k := bytes.NewBufferString(defaultKey)

	go func() {
		log.Printf("Binding HTTP to %v\n", listenPort)
		s := http.Server{
			Addr:         fmt.Sprintf(":%v", listenPort),
			Handler:      r,
			ReadTimeout:  defaultReadTimeout,
			WriteTimeout: defaultWriteTimeout,
		}
		log.Fatal(s.ListenAndServe())
	}()

	log.Println("Binding TLS to ", tlsListAddr)
	log.Fatal(ListenAndServeTLS(s, c, k, r, tlsListAddr))
}

func ListenAndServeTLS(s http.Server, c *bytes.Buffer, k *bytes.Buffer, h http.Handler, listenAddr string) (err error) {
	config := &tls.Config{}
	config.NextProtos = []string{"http/1.1"}

	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.X509KeyPair(c.Bytes(), k.Bytes())
	if err != nil {
		return err
	}

	var ln net.Listener
	ln, err = net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(ln, config)

	return s.Serve(tlsListener)
}
