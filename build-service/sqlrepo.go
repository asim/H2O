package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/hailo-platform/H2O/build-service/models"
)

type sqlRepo struct {
	db     *sql.DB
	dbName string

	getAll           *sql.Stmt
	getAllWithName   *sql.Stmt
	getVersion       *sql.Stmt
	deleteVersion    *sql.Stmt
	getNames         *sql.Stmt
	getCoverage      *sql.Stmt
	getCoverageTrend *sql.Stmt

	createBuild      *sql.Stmt
	addCoverage      *sql.Stmt
	addDependency    *sql.Stmt
	setMergeBaseDate *sql.Stmt
}

// Connect and check that the connection was succesful
// Also prepares the statements
func (r *sqlRepo) Connect(server, port, username, password, dbName string) error {
	var err error

	log.Println("Opening DB")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, server, port, dbName)
	r.db, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("Error opening DB: %v", err)
	}
	log.Println("OK")

	log.Println("Testing DB connection")
	err = r.db.Ping()
	if err != nil {
		return fmt.Errorf("Error connecting to DB: %v", err)
	}
	log.Println("OK")

	r.dbName = dbName

	// Prepare statements
	err = r.prepareStatements()
	if err != nil {
		return err
	}

	return nil
}

func (r *sqlRepo) prepareStatements() (err error) {
	if r.getAll, err = r.db.Prepare("SELECT b.hostname,b.architecture,b.goversion,b.sourceurl,b.binaryurl,b.version,b.language,b.name,b.branch,b.timestamp,c.package,c.percentage,d.importpath,d.commit,d.mergebasedate FROM (SELECT * FROM builds ORDER BY timestamp DESC LIMIT ?) b LEFT JOIN coverage c ON b.name = c.service AND b.version = c.version LEFT JOIN dependencies d ON b.name = d.service AND b.version = d.version"); err != nil {
		return err
	}
	if r.getAllWithName, err = r.db.Prepare("SELECT b.hostname,b.architecture,b.goversion,b.sourceurl,b.binaryurl,b.version,b.language,b.name,b.branch,b.timestamp,c.package,c.percentage,d.importpath,d.commit,d.mergebasedate FROM (SELECT * FROM builds WHERE name=? ORDER BY timestamp DESC LIMIT ?) b LEFT JOIN coverage c ON b.name = c.service AND b.version = c.version LEFT JOIN dependencies d ON b.name = d.service AND b.version = d.version"); err != nil {
		return err
	}
	if r.getVersion, err = r.db.Prepare("SELECT b.hostname,b.architecture,b.goversion,b.sourceurl,b.binaryurl,b.version,b.language,b.name,b.branch,b.timestamp,c.package,c.percentage,d.importpath,d.commit,d.mergebasedate FROM builds b LEFT JOIN coverage c ON b.name = c.service AND b.version = c.version LEFT JOIN dependencies d ON b.name = d.service AND b.version = d.version WHERE b.name=? AND b.version=? ORDER BY b.timestamp DESC"); err != nil {
		return err
	}
	if r.deleteVersion, err = r.db.Prepare("DELETE FROM builds WHERE name=? AND version=?"); err != nil {
		return err
	}
	if r.getNames, err = r.db.Prepare("SELECT DISTINCT name FROM builds WHERE name LIKE ? ORDER BY name ASC"); err != nil {
		return err
	}
	if r.getCoverage, err = r.db.Prepare("SELECT package, ROUND(percentage,2) FROM coverage WHERE service=? AND version=? ORDER BY package ASC"); err != nil {
		return err
	}
	if r.getCoverageTrend, err = r.db.Prepare("SELECT c.service, c.version, b.branch, c.package, ROUND(c.percentage,2), b.timestamp FROM coverage c LEFT JOIN builds b ON b.name = c.service AND b.version = c.version WHERE c.service=? AND timestamp>? ORDER BY b.timestamp ASC, c.package ASC"); err != nil {
		return err
	}

	if r.createBuild, err = r.db.Prepare("INSERT INTO builds (hostname,architecture,goversion,sourceurl,binaryurl,version,language,name,branch,timestamp) VALUES (?,?,?,?,?,?,?,?,?,?)"); err != nil {
		return err
	}
	if r.addCoverage, err = r.db.Prepare("INSERT INTO coverage (service,version,package,percentage) VALUES (?,?,?,?)"); err != nil {
		return err
	}
	if r.addDependency, err = r.db.Prepare("INSERT INTO dependencies (service,version,importpath,commit) VALUES (?,?,?,?)"); err != nil {
		return err
	}
	if r.setMergeBaseDate, err = r.db.Prepare("UPDATE dependencies SET mergebasedate=? WHERE service=? AND version=? AND importpath=? AND commit=?"); err != nil {
		return err
	}
	return nil
}

func (r *sqlRepo) CreateTables() error {
	if _, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS builds (
		  id int(11) unsigned NOT NULL AUTO_INCREMENT,
		  hostname varchar(255) NOT NULL DEFAULT '',
		  architecture varchar(10) NOT NULL DEFAULT '',
		  goversion varchar(255) DEFAULT NULL,
		  sourceurl varchar(255) NOT NULL DEFAULT '',
		  binaryurl varchar(255) NOT NULL DEFAULT '',
		  version varchar(32) NOT NULL DEFAULT '',
		  language varchar(127) NOT NULL DEFAULT '',
		  name varchar(255) NOT NULL DEFAULT '',
		  branch varchar(255) DEFAULT NULL,
		  timestamp bigint(20) unsigned NOT NULL,
		  PRIMARY KEY (id),
		  INDEX idx_name_version (name,version),
		  INDEX idx_timestamp (timestamp)
		) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8;
	`); err != nil {
		return err
	}

	if _, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS coverage (
		  service varchar(255) NOT NULL DEFAULT '',
		  version varchar(32) NOT NULL DEFAULT '',
		  package varchar(255) NOT NULL DEFAULT '',
		  percentage float(5,2) unsigned NOT NULL DEFAULT 000.00,
		  PRIMARY KEY (service,version,package)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8;
	`); err != nil {
		return err
	}

	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS dependencies (
		  service varchar(255) NOT NULL DEFAULT '',
		  version varchar(32) NOT NULL DEFAULT '',
		  importpath varchar(255) NOT NULL DEFAULT '',
		  commit varchar(255) NOT NULL DEFAULT '',
		  mergebasedate bigint(20) unsigned,
		  PRIMARY KEY (service,version,importpath),
		  INDEX idx_importpath_commit (importpath,commit)
		  INDEX idx_importpath_timestamp (importpath,timestamp)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8;
	`)

	return err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

type buildWithJoins struct {
	models.Build
	PackageName   sql.NullString
	Percentage    sql.NullFloat64
	ImportPath    sql.NullString
	Commit        sql.NullString
	MergeBaseDate sql.NullInt64
}

func buildFromRow(rows rowScanner) (*buildWithJoins, error) {
	b := new(buildWithJoins)
	err := rows.Scan(&b.Hostname, &b.Architecture, &b.GoVersion, &b.SourceURL, &b.BinaryURL, &b.Version, &b.Language, &b.Name, &b.Branch, &b.TimeStamp, &b.PackageName, &b.Percentage, &b.ImportPath, &b.Commit, &b.MergeBaseDate)
	return b, err
}

func buildsFromQuery(f func() (*sql.Rows, error)) ([]*models.Build, error) {
	rows, err := f()
	if err != nil {
		return nil, err
	}

	builds := make([]*models.Build, 0)
	buildByName := map[string]*models.Build{}

	for rows.Next() {
		b, err := buildFromRow(rows)
		if err != nil {
			return nil, err
		}

		key := b.Name + b.Version
		build, ok := buildByName[key]
		if !ok {
			build = &b.Build
			build.Coverage = map[string]float64{}
			build.Dependencies = map[string]string{}
			build.MergeBaseDates = map[string]time.Time{}
			buildByName[key] = build
			builds = append(builds, build)
		}

		if b.PackageName.Valid {
			build.Coverage[b.PackageName.String] = b.Percentage.Float64
		}

		if b.ImportPath.Valid {
			build.Dependencies[b.ImportPath.String] = b.Commit.String

			if b.MergeBaseDate.Valid {
				build.MergeBaseDates[b.ImportPath.String] = time.Unix(b.MergeBaseDate.Int64, 0)
			}
		}
	}

	return builds, nil
}

func (r *sqlRepo) Create(b *models.Build) error {
	if _, err := r.createBuild.Exec(
		b.Hostname,
		b.Architecture,
		b.GoVersion,
		b.SourceURL,
		b.BinaryURL,
		b.Version,
		b.Language,
		b.Name,
		b.Branch,
		b.TimeStamp,
	); err != nil {
		return err
	}

	if b.Coverage != nil {
		for packageName, coveragePercentage := range b.Coverage {
			if _, err := r.addCoverage.Exec(b.Name, b.Version, packageName, coveragePercentage); err != nil {
				return err
			}
		}
	}

	if b.Dependencies != nil {
		for importPath, commit := range b.Dependencies {
			if _, err := r.addDependency.Exec(b.Name, b.Version, importPath, commit); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *sqlRepo) SetMergeBaseDate(service, version, importPath, commit string, date time.Time) error {
	_, err := r.setMergeBaseDate.Exec(date.Unix(), service, version, importPath, commit)
	return err
}

func (r *sqlRepo) GetNames(filter string) ([]string, error) {
	names := make([]string, 0)

	rows, err := r.getNames.Query("%" + filter + "%")
	if err != nil {
		return names, err
	}

	for rows.Next() {
		name := new(string)
		err = rows.Scan(name)
		if err != nil {
			return names, err
		}
		names = append(names, *name)
	}

	return names, nil
}

func (r *sqlRepo) GetAll(limit int) ([]*models.Build, error) {
	return buildsFromQuery(func() (*sql.Rows, error) { return r.getAll.Query(limit) })
}

func (r *sqlRepo) GetAllWithName(name string, limit int) ([]*models.Build, error) {
	return buildsFromQuery(func() (*sql.Rows, error) { return r.getAllWithName.Query(name, limit) })
}

func (r *sqlRepo) GetVersion(name, version string) (*models.Build, error) {
	builds, err := buildsFromQuery(func() (*sql.Rows, error) { return r.getVersion.Query(name, version) })
	if len(builds) > 0 {
		return builds[0], err
	}
	return nil, err
}

func (r *sqlRepo) Delete(name, version string) error {
	_, err := r.deleteVersion.Exec(name, version)
	return err
}

func (r *sqlRepo) GetCoverage(name, version string) (map[string]float64, error) {
	rows, err := r.getCoverage.Query(name, version)
	if err != nil {
		return nil, err
	}

	coverage := make(map[string]float64)
	packageName := new(string)
	coveragePercentage := new(float64)

	for rows.Next() {
		err = rows.Scan(packageName, coveragePercentage)
		if err != nil {
			return nil, err
		}
		coverage[*packageName] = *coveragePercentage
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return coverage, nil
}

type coverageRow struct {
	service    string
	version    string
	branch     string
	pkg        string
	percentage float64
	timestamp  int64
}

func groupCoverageRows(coverageRows []coverageRow) models.CoverageSnapshots {
	coverageByTimestamp := make(map[int64][]coverageRow)
	for i, cr := range coverageRows {
		if coverageByTimestamp[cr.timestamp] == nil {
			coverageByTimestamp[cr.timestamp] = make([]coverageRow, 0)
		}
		coverageByTimestamp[cr.timestamp] = append(coverageByTimestamp[cr.timestamp], coverageRows[i])
	}

	snapshots := make(models.CoverageSnapshots, len(coverageByTimestamp))
	i := 0
	for ts, cr := range coverageByTimestamp {
		snapshot := models.CoverageSnapshot{
			Timestamp: ts,
			Coverages: make([]models.Coverage, len(cr)),
		}
		for j, c := range cr {
			if j == 0 {
				snapshot.Branch = c.branch
				snapshot.Version = c.version
			}
			snapshot.Coverages[j] = models.Coverage{
				PackageName: c.pkg,
				Percentage:  c.percentage,
			}
		}

		snapshots[i] = snapshot
		i++
	}

	sort.Sort(snapshots)

	return snapshots
}

func (r *sqlRepo) GetCoverageTrend(name string, since time.Time) (models.CoverageSnapshots, error) {
	rows, err := r.getCoverageTrend.Query(name, since.Unix())
	if err != nil {
		return nil, err
	}

	coverageRows := make([]coverageRow, 0)
	for rows.Next() {
		cr := coverageRow{}
		err = rows.Scan(&cr.service, &cr.version, &cr.branch, &cr.pkg, &cr.percentage, &cr.timestamp)
		if err != nil {
			return nil, err
		}
		coverageRows = append(coverageRows, cr)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	snapshots := groupCoverageRows(coverageRows)

	return snapshots, nil
}
