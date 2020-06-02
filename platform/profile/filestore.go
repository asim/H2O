package profile

import (
	"io"
	"io/ioutil"
	"os"
)

type fileStore struct {
}

func NewFileStore() (fileStore, error) {
	store := fileStore{}

	return store, nil
}

func (s fileStore) Save(id string, reader io.Reader, _ string) (string, error) {
	f, err := ioutil.TempFile(os.TempDir(), id)
	defer f.Close()

	_, err = io.Copy(f, reader)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}
