package config

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	sjson "github.com/bitly/go-simplejson"
	log "github.com/cihub/seelog"

	distance "github.com/HailoOSS/go-distance"
	"github.com/HailoOSS/service/encryption"
)

// DefaultInstance is the default config instance
var DefaultInstance *Config = New()

// Load will load config from a Reader into the default instance
func Load(r io.Reader) error {
	return DefaultInstance.Load(r)
}

// AtPath is a wrapper around DefaultInstance.AtPath
func AtPath(path ...string) ConfigElement {
	return DefaultInstance.AtPath(path...)
}

// AtHob reads hob config in a struct. Returns error if config is missing.
func AtHob(hob string, configStruct interface{}) error {
	atPath := DefaultInstance.AtPath("hobs", hob)

	if string(atPath.AsJson()) == "null" {
		return MissingConfigError{"hob", fmt.Sprintf("hobs/%s", hob)}
	}

	return atPath.AsStruct(configStruct)
}

// AtServiceType reads service type config in a struct. Returns error if config is missing.
func AtServiceType(hob string, serviceType string, configStruct interface{}) error {
	atPath := DefaultInstance.AtPath("serviceTypes", hob, serviceType)

	if string(atPath.AsJson()) == "null" {
		return MissingConfigError{"serviceType", fmt.Sprintf("serviceTypes/%s/%s", hob, serviceType)}
	}

	return atPath.AsStruct(configStruct)
}

// AtServiceTypes reads the service config for all service types for a hob.
// configStruct is expected to be a map[string] -> serviceType config struct
func AtServiceTypes(hob string, configStructs interface{}) error {
	atPath := DefaultInstance.AtPath("serviceTypes", hob)

	if string(atPath.AsJson()) == "null" {
		return MissingConfigError{"serviceTypes", fmt.Sprintf("serviceTypes/%s", hob)}
	}

	return atPath.AsStruct(configStructs)
}

// SubscribeChanges is a wrapper around DefaultInstance.SubscribeChanges
func SubscribeChanges() <-chan bool {
	return DefaultInstance.SubscribeChanges()
}

// LastLoaded wraps DefaultInstance.LastLoaded
func LastLoaded() (string, time.Time) {
	return DefaultInstance.LastLoaded()
}

// Raw wraps DefaultInstance.Raw
func Raw() []byte {
	return DefaultInstance.Raw()
}

// WaitUntilLoaded wraps DefaultInstance.WaitUntilLoaded
func WaitUntilLoaded(d time.Duration) bool {
	return DefaultInstance.WaitUntilLoaded(d)
}

// WaitUntilReloaded wraps DefaultInstance.WaitUntilReloaded
func WaitUntilReloaded(d time.Duration) bool {
	return DefaultInstance.WaitUntilReloaded(d)
}

// configData represents loaded configuration, both raw and parsed. It's only used internally by Config, but as this
// data changes as an atomic unit (literally using an atomic update), it's bundled together.
type configData struct {
	body      *sjson.Json
	raw       []byte
	decrypted map[uint64]*sjson.Json // Cache of decrypted config
	timestamp time.Time
	hash      string
}

// Config represents a bunch of config settings
type Config struct {
	// Used for decrypting secrets
	Service, Region, Env string
	Encryptor            encryption.Encryptor

	data         unsafe.Pointer // *configData -- currently loaded config (may NEVER be nil)
	dataMtx      sync.Mutex     // ensures there are no competing reloads; is not used for reads at all
	observers    []chan bool
	observersMtx sync.RWMutex
}

// unmarshal accepts JSON-encoded bytes and unmarshals this into the config instance
func (c *Config) unmarshal(bytes []byte) (bool, error) {
	newData := &configData{
		body:      new(sjson.Json),
		raw:       bytes,
		decrypted: make(map[uint64]*sjson.Json),
	}
	if err := newData.body.UnmarshalJSON(newData.raw); err != nil {
		return false, err
	}

	h := md5.New()
	h.Write(bytes)
	newData.hash = fmt.Sprintf("%x", h.Sum(nil))

	c.dataMtx.Lock()
	defer c.dataMtx.Unlock()
	currentData := (*configData)(atomic.LoadPointer(&c.data))
	hashChanged := newData.hash != currentData.hash
	if hashChanged {
		newData.timestamp = time.Now()
		atomic.StorePointer(&c.data, (unsafe.Pointer)(newData))
		currentData = newData
	}

	// Clean the decrypted data cache if it is not empty, this should happen
	// even if the data has not changed
	if len(currentData.decrypted) > 0 {
		currentData.decrypted = make(map[uint64]*sjson.Json)
		atomic.StorePointer(&c.data, (unsafe.Pointer)(currentData))
	}

	return hashChanged, nil
}

// AddValidator adds a config validation function to a slice of validators
// deprecated
func (c *Config) AddValidator(v Validator) {}

// Load will load config from a Reader into c
func (c *Config) Load(r io.Reader) error {
	bytes, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("Unable to read config: %v", err)
	}

	hashChanged, err := c.unmarshal(bytes)
	if err != nil {
		return fmt.Errorf("Unable to unmarshal config: %v", err)
	}

	if !hashChanged {
		return nil
	}

	// Config actually changed... log this and notify observers
	data := (*configData)(atomic.LoadPointer(&c.data))
	log.Infof("[Config] Initialised config, loaded hash: %s", data.hash)

	// Notify observers
	c.observersMtx.RLock()
	defer c.observersMtx.RUnlock()
	for _, ch := range c.observers {
		// Non-blocking send
		select {
		case ch <- true:
		default:
		}
	}

	return nil
}

// AtPath will get a ConfigElement at the specified path
func (c *Config) AtPath(path ...string) ConfigElement {
	data := (*configData)(atomic.LoadPointer(&c.data))
	return &JSONElement{c, data, data.body.GetPath(path...)}
}

// SubscribeChanges will yield a channel which will then receive a boolean whenever
// the loaded configuration changes (depending on the exact loader used)
func (c *Config) SubscribeChanges() <-chan bool {
	c.observersMtx.Lock()
	defer c.observersMtx.Unlock()

	ch := make(chan bool, 1)
	c.observers = append(c.observers, ch)

	return (<-chan bool)(ch)
}

// LastLoaded will return the time we last loaded config, along with the hash
func (c *Config) LastLoaded() (string, time.Time) {
	data := (*configData)(atomic.LoadPointer(&c.data))
	return data.hash, data.timestamp
}

// WaitUntilLoaded waits for a maximum amount of duration d for the config to be successfully loaded. The idea is that
// we would prefer to soldier on if we cannot load config, but we don't mind delaying service boot times a little bit
// if it means they start off with some config loaded
func (c *Config) WaitUntilLoaded(d time.Duration) bool {
	ch := c.SubscribeChanges()
	_, t := c.LastLoaded()
	if !t.IsZero() {
		return true // already loaded
	}

	log.Tracef("[Config] Waiting for %v until config loaded…", d)
	select {
	case <-ch:
	case <-time.After(d):
	}

	// Double-check
	_, t = c.LastLoaded()
	return !t.IsZero()
}

// WaitUntilReloaded waits for a maximum amount of duration d for the config to be successfully loaded. This contrasts
// with WaitUntilLoaded in that it only returns after the config has been refreshed; it will not return immediately in
// the case that config is already cached
func (c *Config) WaitUntilReloaded(d time.Duration) bool {
	ch := c.SubscribeChanges()

	log.Tracef("[Config] Waiting for %v until config reloaded…", d)
	select {
	case <-ch:
	case <-time.After(d):
	}

	_, t := c.LastLoaded()
	return !t.IsZero()
}

// Raw returns entire raw loaded config as bytes
func (c *Config) Raw() []byte {
	data := (*configData)(atomic.LoadPointer(&c.data))
	return data.raw
}

func (c *Config) cacheDecryptedConfig(hashKey uint64, body *sjson.Json) {
	c.dataMtx.Lock()
	defer c.dataMtx.Unlock()

	data := (*configData)(atomic.LoadPointer(&c.data))
	data.decrypted[hashKey] = body
	atomic.StorePointer(&c.data, (unsafe.Pointer)(data))
}

// New mints a new config
func New() *Config {
	return &Config{
		Encryptor: &encryption.KMSEncryptor{
			KMS: kms.New(session.New(), &aws.Config{Region: aws.String(os.Getenv("EC2_REGION"))}),
		},
		data: (unsafe.Pointer)(&configData{
			body:      new(sjson.Json),
			decrypted: make(map[uint64]*sjson.Json),
		}),
		observers: make([]chan bool, 0),
	}
}

// ConfigElement represents some specific piece of config, that we have drilled down to
type ConfigElement interface {
	AsString(def string) string
	AsBool() bool
	AsInt(def int) int
	AsFloat64(def float64) float64
	AsDuration(def string) time.Duration
	AsStringArray() []string
	AsHostnameArray(defPort int) []string
	AsStringMap() map[string]string
	AsStruct(val interface{}) error
	AsJson() []byte
	Decrypt() (ConfigElement, error)
	AtPath(path ...string) ConfigElement
}

// JSONElement is the default implementation of ConfigElement
type JSONElement struct {
	config     *Config     // Reference parent config
	configData *configData // Reference parent data (used to cache decrypted secrets)

	*sjson.Json
}

// AsString will retrieve a single config value as a string. It will return
// a blank string if there is no value corresponding to the supplied path, or
// alternatively the supplied default value.
func (c *JSONElement) AsString(def string) string {
	return c.MustString(def)
}

// AsStringArray will retrieve an array of config values, each as a string.
func (c *JSONElement) AsStringArray() []string {
	arr := make([]string, 0)
	genericValues, err := c.Array()
	if err != nil {
		return arr
	}

	// assert every single value of the array, construct our
	for _, value := range genericValues {
		switch value.(type) {
		case string:
			arr = append(arr, value.(string))
		}
	}

	return arr
}

// AsStringMap will retrieve a map of string config values with the children
// of the specified path being string keys to their descendents.
func (c *JSONElement) AsStringMap() map[string]string {
	results := make(map[string]string)

	for key, val := range c.MustMap() {
		skey := fmt.Sprintf("%v", key)
		sval := fmt.Sprintf("%v", val)
		results[skey] = sval
	}

	return results
}

// AsHostnameArray will retrieve an array of config values, where each one
// is a string made up of a hostname:port. Any values defined in config
// without a :port bit will have this automatically added.
func (c *JSONElement) AsHostnameArray(defPort int) []string {
	arr := c.AsStringArray()
	for i, value := range arr {
		parts := strings.Split(value, ":")
		if len(parts) == 1 {
			arr[i] = fmt.Sprintf("%s:%v", value, defPort)
		}
	}

	return arr
}

// AsBool will retrieve a single config value as a string. It works with our
// config service and will automatically interpret a string of "true" as a
// boolean true, and treat undefined as false.
func (c *JSONElement) AsBool() bool {
	value := false

	if v, err := c.Bool(); err == nil {
		value = v
	} else if v, err := c.Json.String(); err == nil {
		if v == "true" || v == "1" {
			value = true
		}
	} else if v, err := c.Json.Int(); err == nil {
		if v == 1 {
			value = true
		}
	}

	return value
}

// AsInt will retrieve a single config value as an integer.
func (c *JSONElement) AsInt(def int) (value int) {
	value, err := c.Json.Int()
	if err != nil {
		return def
	}

	return value
}

// AsFloat64 will retrieve a single config value as a float.
func (c *JSONElement) AsFloat64(def float64) (value float64) {
	value, err := c.Json.Float64()
	if err != nil {
		return def
	}

	return value
}

// AsDuration will retrieve a single config value as a duration, parsing a
// string like "10ms" or "5s" etc.
func (c *JSONElement) AsDuration(def string) (value time.Duration) {
	if v, err := c.Json.String(); err == nil {
		if value, err := time.ParseDuration(v); err == nil {
			return value
		}
	}

	// use the default - if it doesn't parse then return 0 duration
	value, err := time.ParseDuration(def)
	if err != nil {
		log.Errorf("[Config] Failed to parse default duration value: %s", def)
	}

	return value
}

// AsStruct will retrieve a single config value, marshaling it into the provided
// empty struct.
func (c *JSONElement) AsStruct(val interface{}) error {
	// @todo is it possible to avoid marshal + unmarshal step?
	b, err := c.Json.MarshalJSON()

	if err != nil {
		return fmt.Errorf("Error finding bytes in config: %v", err)
	}
	if err := json.Unmarshal(b, val); err != nil {
		return fmt.Errorf("Error unmarshaling to struct: %v", err)
	}

	return nil
}

// AsJson will retrieve a single config value, as JSON-encoded data in byte form.
func (c *JSONElement) AsJson() []byte {
	b, _ := c.Json.MarshalJSON()

	return b
}

// AtPath will get a ConfigElement at the specified path
func (c *JSONElement) AtPath(path ...string) ConfigElement {
	return &JSONElement{c.config, c.configData, c.GetPath(path...)}
}

func (c *JSONElement) Decrypt() (ConfigElement, error) {
	ctx := map[string]string{
		"service-name": c.config.Service,
	}

	return c.DecryptWithContext(ctx)
}

func (c *JSONElement) DecryptWithContext(ctx map[string]string) (ConfigElement, error) {
	encodedData, err := c.String()
	if err != nil {
		return nil, err
	}

	// First check cache for decrypted data
	h := fnv.New64a()
	h.Write([]byte(encodedData))
	hashKey := h.Sum64()

	var data *sjson.Json
	var ok bool
	data, ok = c.configData.decrypted[hashKey]
	// If data was not found then decrypt the data
	if !ok {
		// Encrypted config is stored as a base64 encoded string so first decode the
		// string
		encryptedData, err := base64.StdEncoding.DecodeString(encodedData)
		if err != nil {
			return nil, err
		}

		ctx["region"] = c.config.Region
		ctx["environment"] = c.config.Env

		// Decrypt data using the built context
		body, err := c.config.Encryptor.Decrypt(ctx, encryptedData)
		if err != nil {
			return nil, err
		}

		data, err = sjson.NewJson(body)
		if err != nil {
			return nil, err
		}

		// Store decrypted data in the parent configData
		c.config.cacheDecryptedConfig(hashKey, data)

	}

	return &JSONElement{c.config, c.configData, data}, nil
}

// AsDistance will retrieve a single config value as a distance, parsing a
// string like "10mi" or "5km" etc.
func (c *JSONElement) AsDistance(def string) (value distance.Distance) {
	if v, err := c.Json.String(); err != nil {
		if value, err := distance.ParseDistance(v); err == nil {
			return value
		}
	}

	// use the default - if it doesn't parse then return 0 distance
	value, err := distance.ParseDistance(def)
	if err != nil {
		log.Errorf("[Config] Failed to parse default distance value: %s", def)
	}

	return value
}

type MissingConfigError struct {
	configType string
	path       string
}

func (e MissingConfigError) Error() string {
	if e.configType != "" {
		return fmt.Sprintf("Missing %s config at %s", e.configType, e.path)
	} else {
		return fmt.Sprintf("Missing config at %s", e.path)
	}
}
