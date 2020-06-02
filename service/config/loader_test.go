package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReaderLoader(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService": {"hash": {"alpha": "a", "num": 1}}}`)
	Load(buf)
	if s := AtPath("configService", "hash", "alpha").AsString(""); s != "a" {
		t.Errorf("Expecting a, got %v", s)
	}
}

func TestSubscribeChanges(t *testing.T) {
	setupTest()

	ch := SubscribeChanges()

	updateChan := make(chan bool)
	go func() {
		var gotUpdate bool
		select {
		case gotUpdate = <-updateChan:
		case updateChan <- gotUpdate:
		}
	}()

	updateChan <- false
	go func() {
		<-ch
		updateChan <- true
	}()

	buf := bytes.NewBufferString(`{"configService": {"hash": {"alpha": "a", "num": 1}}}`)
	Load(buf)

	assert.True(t, <-updateChan, "Expecting to receive update notifcation on SubscribeChanges channel")
}

func TestSubscribeChangesTimesoutIfNoListener(t *testing.T) {
	setupTest()

	SubscribeChanges()
	buf := bytes.NewBufferString(`{"configService": {"hash": {"alpha": "a", "num": 1}}}`)

	// run loader in goroutine so we can timeout and fail this test
	done := make(chan bool)
	timeout := time.After(5 * time.Second)
	go func() {
		Load(buf)
		done <- true
	}()

	select {
	case <-done:
	case <-timeout:
		assert.Fail(t, "Expecting update notifcation to timeout when unwatched channel")
	}
}

func TestLoadFromFile(t *testing.T) {
	fh, err := ioutil.TempFile("", "")
	assert.NoError(t, err, "Failed to create temporary file")
	defer os.Remove(fh.Name())

	fn := fh.Name()
	_, err = fh.WriteString(`{"configService": {"hash": {"dave": "bazzzz", "foo": "barrrr"}}}`)
	assert.NoError(t, err, "Failed to write to temporary file")
	fh.Close()

	// do load now
	LoadFromFile(fn)
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, "barrrr", AtPath("configService", "hash", "foo").AsString("nada"))

	// now change the file
	err = ioutil.WriteFile(fn, []byte(`{"configService": {"hash": {"dave": "boom", "foo": "bing"}}}`), os.ModePerm)
	assert.NoError(t, err, "Failed to update temporary file")

	// Now wait for the grace period, for the changes to be picked up
	assert.True(t, WaitUntilReloaded(11*time.Second), "Config failed to reload in grace period")

	// now verify the change
	assert.Equal(t, "bing", AtPath("configService", "hash", "foo").AsString("nada"), "Unexpected second read result")
}

// Check that when the default loader is replaced (by calling LoadFromFile or LoadFromService), the old one is killed
func TestDefaultLoaderKilledWhenReplaced(t *testing.T) {
	fh, err := ioutil.TempFile("", "")
	if err != nil {
		t.Errorf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(fh.Name())

	fn := fh.Name()
	_, err = fh.WriteString(`{"configService": {"oliver": "beattie"}}`)
	assert.NoError(t, err, "Error writing to temporary file")
	fh.Close()

	err = LoadFromFile(fn)
	assert.NoError(t, err)
	loader := defaultLoader

	err = LoadFromFile(fn)
	assert.NoError(t, err)
	newLoader := defaultLoader
	assert.NotEqual(t, loader, newLoader, "defaultLoader should have been replaced")
	select {
	case <-loader.Dead():
	case <-time.After(10 * time.Second):
		assert.Fail(t, "Old defaultLoader failed to die")
	}

	// Check the new loader is not killed
	select {
	case <-newLoader.Dead():
		assert.Fail(t, "New defaultLoader died too (it shouldn't have)")
	default:
	}

	SetLoader(new(Loader))
	newNewLoader := defaultLoader
	defer newNewLoader.Killf("Test exiting")
	assert.NotEqual(t, newLoader, newNewLoader, "defaultLoader should have been replaced")
	select {
	case <-newLoader.Dead():
	case <-time.After(10 * time.Second):
		assert.Fail(t, "Old defaultLoader failed to die")
	}
}
