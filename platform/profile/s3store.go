package profile

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/hailo-platform/H2O/goamz/aws"
	"github.com/hailo-platform/H2O/goamz/s3"
)

type s3Store struct {
	auth   aws.Auth
	bucket *s3.Bucket
}

func NewS3Store(bucket string) (s3Store, error) {
	store := s3Store{}

	var err error
	store.auth, err = aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return store, err
	}

	sss := s3.New(store.auth, aws.EUWest)
	store.bucket = sss.Bucket(bucket)

	return store, nil
}

func (s s3Store) Save(id string, reader io.Reader, contentType string) (string, error) {
	// We need to buffer the data since S3 does not support sending streams
	// without a content-length
	// http://aws.amazon.com/articles/1109?_encoding=UTF8&jiveRedirect=1#14
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, reader)
	if err != nil {
		return "", err
	}

	err = s.bucket.Put(id, buf.Bytes(), contentType, "", s3.Options{})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("s3://%s/%s", s.bucket.Name, id), nil
}
