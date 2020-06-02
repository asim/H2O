package config

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/jpillora/backoff"
	"gopkg.in/tomb.v1"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	// Worst-case config loader interval -- hard attempt to reload config
	configPollInterval = 30 * time.Minute
	// For file loader, how often to inspect mtime
	filePollInterval = 10 * time.Second
	// Sleep between load failure
	configMinRetryDelay    = 1 * time.Minute
	configMaxRetryDelay    = 10 * time.Minute
	configRetryDelayFactor = 2
	// The maximum number of times to attempt to reload the config
	configMaxRetryAttempts = 4
)

var defaultLoader *Loader

type reader func() (io.ReadCloser, error)

// Loader represents a loader of configuration. It automatically reloads when it receives notification that it should
// do so on its changes channel, and also every configPollInterval
type Loader struct {
	tomb.Tomb
	c          *Config
	changes    <-chan bool
	r          reader
	reloadLock sync.Mutex
	backoff    *backoff.Backoff
}

// Load will go and grab the config via the reader and then load it into the config
func (ldr *Loader) Load() error {
	r, err := ldr.r()
	if err != nil {
		return err
	}
	// make sure we close this read-closer that we've just got, once we're done with it
	defer r.Close()
	err = ldr.c.Load(r)
	if err != nil {
		return err
	}

	return nil
}

func (ldr *Loader) Reload() {
	ldr.reloadLock.Lock()
	defer ldr.reloadLock.Unlock()

	for {
		if err := ldr.Load(); err != nil {
			log.Warnf("[Config] Failed to reload config: %v", err)
			// We backoff here since config-reload events trigger Reloads
			// at similar times and spam the config-service
			if ldr.backoff.Attempt() > configMaxRetryAttempts {
				break
			}
			time.Sleep(ldr.backoff.Duration())
			continue
		}
		break
	}

	ldr.backoff.Reset()
}

func NewLoader(c *Config, changes chan bool, r reader) *Loader {
	ldr := &Loader{
		c:       c,
		changes: changes,
		r:       r,
		backoff: &backoff.Backoff{
			Min:    configMinRetryDelay,
			Max:    configMaxRetryDelay,
			Factor: configRetryDelayFactor,
			Jitter: true,
		},
	}

	go func() {
		defer ldr.Done()

		tick := time.NewTicker(configPollInterval)
		defer tick.Stop()

		for {
			select {
			case <-changes:
				ldr.Reload()
			case <-tick.C:
				ldr.Reload()
			case <-ldr.Dying():
				log.Tracef("[Config] Loader dying: %s", ldr.Err().Error())
				return
			}
		}
	}()

	// Spit a change down the pipe to load now
	changes <- true

	return ldr
}

// NewFileLoader returns a loader that reads config from file fn
func NewFileLoader(c *Config, fn string) (*Loader, error) {
	log.Infof("[Config] Initialising config loader to load from file '%s'", fn)

	rdr := func() (io.ReadCloser, error) {
		file, err := os.Open(fn)
		if err != nil {
			return nil, fmt.Errorf("Error opening file %v: %v", fn, err)
		}
		return file, nil
	}

	changesChan := make(chan bool)
	l := NewLoader(c, changesChan, rdr)
	go func() {
		tick := time.NewTicker(filePollInterval)
		defer tick.Stop()

		var lastMod time.Time
		for {
			select {
			case <-tick.C:
				if fi, err := os.Stat(fn); err == nil {
					if fi.ModTime().After(lastMod) {
						lastMod = fi.ModTime()
						changesChan <- true
					}
				}
			case <-l.Dying():
				// When the loader dies, we should too
				return
			}
		}
	}()

	return l, nil
}

// LoadFromFile will load config from a flat text file containing JSON into the default instance
func LoadFromFile(fn string) (err error) {
	if defaultLoader != nil {
		defaultLoader.Killf("Replaced by LoadFromFile")
	}

	defaultLoader, err = NewFileLoader(DefaultInstance, fn)
	return err
}

// LoadFromService will load config from the config service into the default instance
func SetLoader(loader *Loader) {
	if defaultLoader != nil {
		defaultLoader.Killf("Replaced by SetLoader")
	}

	defaultLoader = loader
}
