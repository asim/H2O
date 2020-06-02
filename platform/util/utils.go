package util

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
	"strings"

	log "github.com/cihub/seelog"
)

// GetAwsAZName returns the AWS availability zone of this machine
func GetAwsAZName() (name string, err error) {
	// On an aws machine just curl
	if bytes, err := ioutil.ReadFile("/etc/h2o/azname"); err != nil {
		log.Criticalf("Couldn't read file /etc/h2o/azname This file should contain the AZ name: %v", err)
	} else {
		name = strings.TrimSpace(string(bytes))
	}

	return
}

// GetMD5Hash creates an MD5 hash
func GetMD5Hash(bytes []byte) string {
	h := md5.New()
	h.Write(bytes)
	return hex.EncodeToString(h.Sum(nil))
}

// GetEnvironmentName returns the name of the environment this machine is part of
func GetEnvironmentName() string {
	return os.Getenv("H2O_ENVIRONMENT_NAME")
}

// GetAwsRegionName returns the region this server is in
func GetAwsRegionName() string {
	return os.Getenv("EC2_REGION")
}
