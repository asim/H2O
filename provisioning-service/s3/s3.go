package s3

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/HailoOSS/provisioning-service/dao"
	"github.com/HailoOSS/provisioning-service/process"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	buildsBucket   = "hailo-builds"
	requestTimeout = time.Minute
)

var (
	depsBucket = os.Getenv("HAILO_DEPS_BUCKET")
)

type S3Mgr struct {
	buckets map[string]*s3Bucket
	conns   map[string]*s3.S3
}

type s3Bucket struct {
	*s3.Bucket
	name   string
	prefix string
}

func newS3Conn(regionPrefix string) *s3.S3 {
	auth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		log.Errorf("Error calling aws.GetAuth: %v", err)
		return nil
	}

	region := aws.Region{
		Name:       "EU (Ireland) Region",
		S3Endpoint: "https://s3-eu-west-1.amazonaws.com",
	}

	switch regionPrefix {
	case "us":
		region = aws.USEast
	case "eu":
		region = aws.EUWest
	case "ap":
		region = aws.APNortheast
	}

	s3c := s3.New(auth, region)
	s3c.RequestTimeout = requestTimeout
	return s3c
}

func s3Path(ps *dao.ProvisionedService) string {
	return fmt.Sprintf(
		"%v/%v-%v",
		strings.Replace(ps.ServiceName, ".", "/", -1),
		ps.ServiceName,
		ps.ServiceVersion,
	)
}

func New() *S3Mgr {
	return &S3Mgr{
		buckets: make(map[string]*s3Bucket),
		conns:   make(map[string]*s3.S3),
	}
}

func (s *s3Bucket) path(name string) string {
	return filepath.Join(s.prefix, name, "/")
}

// bucket returns an s3 bucket
func (s *S3Mgr) bucket(name string) (*s3Bucket, error) {
	b, ok := s.buckets[name]
	if !ok {
		return nil, fmt.Errorf("Do not have s3 bucket %s", name)
	}
	return b, nil
}

// conn returns an *s3.S3 struct from the conn map if it exists or creates a new struct.
// Region is the aws region prefix us, eu or ap.
func (s *S3Mgr) conn(region string) *s3.S3 {
	r := "default"

	switch region {
	case "us", "eu", "ap":
		r = region
	}

	c, ok := s.conns[r]
	if !ok {
		c = newS3Conn(r)
		s.conns[r] = c
	}

	return c
}

func getRegionFromBucket(b string) string {
	// Expected format hailo-builds or hailo-deps-eu
	if t := strings.Split(b, "-"); len(t) == 3 {
		return t[2]
	} else {
		return "default"
	}
}

// Setup is not an init() function since it must not be run during the build process
func (s *S3Mgr) Setup() error {
	for name, bucket := range map[string]string{"builds": buildsBucket, "deps": depsBucket} {
		if len(bucket) == 0 {
			panic(fmt.Sprintf("s3 bucket for hailo-%s is undefined", name))
		}
		s3b := strings.Split(bucket, "/")

		region := getRegionFromBucket(s3b[0])
		sthree := s.conn(region)

		s.buckets[bucket] = &s3Bucket{
			sthree.Bucket(s3b[0]),
			s3b[0],
			strings.Join(s3b[1:], "/"),
		}
	}

	return nil
}

// Download will download the binary/JAR for this provisioned service from S3
// and store within the local filesystem, returning the full path to the
// new file
func (s *S3Mgr) Download(ps *dao.ProvisionedService) (string, error) {
	return s.DownloadFile(buildsBucket, s3Path(ps), process.ExePath(ps))
}

// DownloadFile retrieves a file from S3
func (s *S3Mgr) DownloadFile(bucketName, remotePath, localPath string) (string, error) {
	if ok, err := s.FileExists(bucketName, remotePath); !ok {
		return localPath, fmt.Errorf("File does not exist in S3: %v, %v", remotePath, err)
	}

	// make sure folder exists
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, os.ModeDir|os.ModePerm); err != nil {
		return localPath, err
	}

	fh, err := os.Create(localPath)
	if err != nil {
		return localPath, err
	}

	defer fh.Close()

	bucket, err := s.bucket(bucketName)
	if err != nil {
		return localPath, err
	}

	bckR, err := bucket.GetReader(bucket.path(remotePath))
	if err != nil {
		return localPath, err
	}

	if _, err := io.Copy(fh, bckR); err != nil {
		return localPath, err
	}

	if err := os.Chmod(localPath, 0755); err != nil {
		return localPath, err
	}

	return localPath, nil
}

// Exists will check if this provisioned service exists on S3 (this is our
// test of whether it is a valid provisioned service)
func (s *S3Mgr) Exists(ps *dao.ProvisionedService) (bool, error) {
	// using a prefix of our entire name as we expect
	return s.FileExists(buildsBucket, s3Path(ps))
}

// FileExists checks whether a file exists in S3
func (s *S3Mgr) FileExists(bucketName, remotePath string) (bool, error) {
	bucket, err := s.bucket(bucketName)
	if err != nil {
		return false, err
	}

	res, err := bucket.List(bucket.path(remotePath), "", "", 1)
	if err != nil {
		return false, err
	}

	if len(res.Contents) == 1 {
		return true, nil
	}

	return false, nil
}

// IsDownloaded will check if we have already downloaded this binary/JAR to
// the local filesystem, returning the full path to the file
func (s *S3Mgr) IsDownloaded(ps *dao.ProvisionedService) (bool, string) {
	dst := process.ExePath(ps)
	if _, err := os.Stat(dst); err != nil {
		return false, dst
	}

	return true, dst
}

// Delete removes a downloaded file, incase of errors copying
func (s *S3Mgr) Delete(ps *dao.ProvisionedService) error {
	if ok, dst := s.IsDownloaded(ps); ok {
		log.Infof("Removing provisioned binary: %v", ps)
		return os.Remove(dst)
	}

	return nil
}

// VerifyBinary downloads the md5sum for a binary if it exists and checks it against the
// downloaded binary. If the md5sum file does not exist
func (s *S3Mgr) VerifyBinary(ps *dao.ProvisionedService) error {
	binary := process.ExePath(ps)
	remoteMD5 := fmt.Sprintf("%s.md5", s3Path(ps))
	localMD5 := fmt.Sprintf("%s.md5", binary)

	// If the md5 file does not exist just return
	if ok, _ := s.FileExists(buildsBucket, remoteMD5); !ok {
		log.Warnf("Missing remote md5 for %s... ignoring", binary)
		return nil
	}

	// Attempt to download the md5 file
	if _, err := s.DownloadFile(buildsBucket, remoteMD5, localMD5); err != nil {
		return err
	}

	// Get the md5 from the file
	f, err := os.Open(localMD5)
	defer f.Close()
	if err != nil {
		return err
	}
	r := bufio.NewReader(f)
	str, err := r.ReadString('\n')
	if err != nil {
		return err
	}

	// The remote md5
	rmd5 := strings.TrimRight(str, "\n")

	// Get the md5 from the binary
	f2, err := os.Open(binary)
	defer f2.Close()
	if err != nil {
		return err
	}

	h := md5.New()
	_, err = io.Copy(h, f2)
	if err != nil {
		return err
	}

	// The binary md5
	bmd5 := hex.EncodeToString(h.Sum(nil))

	if rmd5 != bmd5 {
		return fmt.Errorf("Failed to verify md5 for %s. Binary %s, remote md5 file %s", binary, bmd5, rmd5)
	}

	log.Debugf("Verified md5 for %s. Binary %s, remote md5 file %s", binary, bmd5, rmd5)
	return nil
}
