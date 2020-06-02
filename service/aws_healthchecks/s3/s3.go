/**
 * This is a healthcheck for the Amazon AWS S3 Service
 */

package aws_healthchecks_s3

import (
	"fmt"
	"github.com/HailoOSS/service/healthcheck"
	"github.com/HailoOSS/goamz/aws"
	"github.com/HailoOSS/goamz/s3"
	"regexp"
	"time"
)

const HealthCheckId = "com.HailoOSS.service.aws_s3"

// HealthCheck asserts we can connect to s3
func HealthCheck(accessKey string, secretKey string, region string) healthcheck.Checker {
	return func() (map[string]string, error) {
		auth := aws.Auth{AccessKey: accessKey, SecretKey: secretKey}
		client := s3.New(auth, aws.Regions[region])

		bucketName := fmt.Sprintf("hailo-healthcheck-bucket-%d", time.Now().UTC().UnixNano())
		b := client.Bucket(bucketName)
		_, err := b.Get("non-existent")

		re := regexp.MustCompile("no such host")

		if re.MatchString(err.Error()) {
			return nil, err
		}

		return nil, nil
	}
}
