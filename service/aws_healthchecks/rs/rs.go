/**
 * This is a healthcheck for the Amazon AWS Redshift Service
 */

package aws_healthchecks_rs

import (
	"database/sql"
	"github.com/HailoOSS/service/healthcheck"
	_ "github.com/lib/pq"
)

const HealthCheckId = "com.HailoOSS.service.aws_rs"

// HealthCheck asserts we can connect to rs
func HealthCheck(odbc string) healthcheck.Checker {
	return func() (map[string]string, error) {
		db, err := sql.Open("postgres", odbc)
		if err != nil {
			return nil, err
		}
		err = db.Ping()
		if err != nil {
			return nil, err
		}

		db.Close()

		return nil, nil
	}
}
