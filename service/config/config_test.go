package config

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/HailoOSS/service/encryption"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type Example struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
	Baz int32  `json:"baz"`
}

func setupTest() {
	DefaultInstance = New()
}

func TestStringMap(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService": {"hash": {"alpha": "a", "num": 1}}}`)
	Load(buf)
	stringMap := AtPath("configService", "hash").AsStringMap()
	expected := map[string]string{"alpha": "a", "num": "1"}
	if !reflect.DeepEqual(expected, stringMap) {
		t.Errorf("Exepecting %v, got %v", expected, stringMap)
	}
}

func TestHostnameArray(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService":{"hash":["a","b","c:8080"]}}`)
	Load(buf)
	a := AtPath("configService", "hash").AsHostnameArray(1024)

	expected := []string{"a:1024", "b:1024", "c:8080"}
	if !reflect.DeepEqual(expected, a) {
		t.Errorf("Exepecting %v, got %v", expected, a)
	}
}

func TestStringArray(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService":{"hash":["a","b","c"]}}`)
	Load(buf)
	a := AtPath("configService", "hash").AsStringArray()

	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(expected, a) {
		t.Errorf("Exepecting %v, got %v", expected, a)
	}
}

func TestInt(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService":{"hash":42}}`)
	Load(buf)
	i := AtPath("configService", "hash").AsInt(10)
	if i != 42 {
		t.Error("Expecting int of 42")
	}
}

func TestIntDefault(t *testing.T) {
	setupTest()
	i := AtPath("configService", "hash").AsInt(10)
	if i != 10 {
		t.Error("Expecting default int of 10")
	}
}

func TestString(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService": {"hash": "foobarbaz"}}`)
	Load(buf)
	s := AtPath("configService", "hash").AsString("default")
	if s != "foobarbaz" {
		t.Error("Expecting string of foobarbaz")
	}
}

func TestMissingStringDefault(t *testing.T) {
	setupTest()
	s := AtPath("configservice", "hash").AsString("default")
	if s != "default" {
		t.Error("Expecting default string of default")
	}
}

func TestBoolBool(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService": {"someBool": true}}`)
	Load(buf)
	if b := AtPath("configService", "someBool").AsBool(); !b {
		t.Error("Expecting someBool to be true")
	}
}

func TestBoolDefault(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService": {"someBool": true}}`)
	Load(buf)
	if b := AtPath("configService", "randomThing").AsBool(); b {
		t.Error("Expecting randomThing to be false")
	}
}

func TestBoolTrueString(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService": {"someBool": "true"}}`)
	Load(buf)
	if b := AtPath("configService", "someBool").AsBool(); !b {
		t.Error("Expecting someBool ('true') to be true")
	}
}

func TestBoolOneString(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService": {"someBool": "1"}}`)
	Load(buf)
	if b := AtPath("configService", "someBool").AsBool(); !b {
		t.Error("Expecting someBool ('1') to be true")
	}
}

func TestBoolOneInt(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"configService": {"someBool": 1}}`)
	Load(buf)
	if b := AtPath("configService", "someBool").AsBool(); !b {
		t.Error("Expecting someBool (1) to be true")
	}
}

func TestDuration(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"foo": {"bar": "100ms"}}`)
	Load(buf)
	v := AtPath("foo", "bar").AsDuration("250ms")
	if v != time.Millisecond*100 {
		t.Error("Expecting duration of 100ms")
	}
}

func TestDurationDefault(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"foo": {"bar": "100ms"}}`)
	Load(buf)
	v := AtPath("foo", "baz").AsDuration("250ms")
	if v != time.Millisecond*250 {
		t.Error("Expecting duration of 250ms")
	}
}

func TestMapStructValid(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"dave": {"a": {"foo":"this-is-foo", "bar":"this-is-bar", "baz":42}, "b": {"foo":"this-is-foo", "bar":"this-is-bar", "baz":42}}}`)
	Load(buf)
	m := make(map[string]Example)
	if err := AtPath("dave").AsStruct(&m); err != nil {
		t.Fatalf("Err unmarshaling JSON into struct map: %v", err)
	}
	if m["a"].Foo != "this-is-foo" {
		t.Error("Struct map does not contain expected foo:\"this-is-foo\", got: ", m["a"].Foo)
	}
}

func TestStructValid(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"dave": {"foo":"this-is-foo", "bar":"this-is-bar", "baz":42}}`)
	Load(buf)
	s := &Example{}
	if err := AtPath("dave").AsStruct(s); err != nil {
		t.Fatalf("Err unmarshaling JSON into struct: %v", err)
	}
	if s.Foo != "this-is-foo" {
		t.Error("Struct does not contain expected foo:\"this-is-foo\"")
	}
}

func TestAsJson(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"dave": {"foo":"this-is-foo", "bar":"this-is-bar", "baz":42}}`)
	Load(buf)
	b := AtPath("dave").AsJson()
	// not convinced this is a sane idea -- can we rely on json being in this order?
	if !bytes.Equal(b, []byte(`{"bar":"this-is-bar","baz":42,"foo":"this-is-foo"}`)) {
		t.Error("Err fetching raw JSON - does not match expected")
	}
}

func TestAtHob(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"hobs": {"ATL": { "key": "value" }, "MNC": {"key": "value2"}}} `)
	Load(buf)

	var config struct {
		Key string `json:"key"`
	}

	err := AtHob("ATL", &config)
	assert.NoError(t, err)
	assert.Equal(t, "value", config.Key)

	err = AtHob("MNC", &config)
	assert.NoError(t, err)
	assert.Equal(t, config.Key, "value2")

	// try reading missing config
	err = AtHob("LIV", &config)
	if err == nil {
		t.Errorf("Error expected")
	}
}

func TestAtServiceType(t *testing.T) {
	setupTest()
	buf := bytes.NewBufferString(`{"serviceTypes": {"ATL": { "regular": {"key": "1"}, "exec": {"key": "2"} }, "MNC": { "regular": {"key": "3"}}}}`)
	Load(buf)

	var config struct {
		Key string `json:"key"`
	}

	err := AtServiceType("ATL", "regular", &config)
	assert.NoError(t, err)
	assert.Equal(t, config.Key, "1")

	err = AtServiceType("ATL", "exec", &config)
	assert.NoError(t, err)
	assert.Equal(t, config.Key, "2")

	err = AtServiceType("MNC", "regular", &config)
	assert.NoError(t, err)
	assert.Equal(t, config.Key, "3")

	err = AtServiceType("ATL", "pimms", &config)
	if err == nil {
		t.Errorf("Error expected")
	}
}

func TestDecrypt(t *testing.T) {
	val := `{"secret": "supersecretstring"}`
	ctx := map[string]string{"service-name": "test", "region": "eu-west-1", "environment": "lve"}

	kmsApi := &encryption.MockKMSAPI{}
	encryptor := &encryption.KMSEncryptor{KMS: kmsApi}
	kmsApi.On("GenerateDataKey", &kms.GenerateDataKeyInput{
		EncryptionContext: map[string]*string{
			"service-name": aws.String("test"),
			"region":       aws.String("eu-west-1"),
			"environment":  aws.String("lve"),
		},
		KeySpec: aws.String("AES_256"),
		KeyId:   aws.String("master-key-id"),
	}).Return(&kms.GenerateDataKeyOutput{
		KeyId:          aws.String("key-id"),
		CiphertextBlob: []byte("ciphertext-blob"),
		Plaintext:      make([]byte, 32),
	}, nil)
	kmsApi.On("Decrypt", mock.AnythingOfType("*kms.DecryptInput")).Return(&kms.DecryptOutput{
		KeyId:     aws.String("key-id"),
		Plaintext: make([]byte, 32),
	}, nil)

	encrypted, err := encryptor.Encrypt("master-key-id", ctx, []byte(val))
	require.Nil(t, err)

	setupTest()
	DefaultInstance.Encryptor = encryptor
	DefaultInstance.Service = "test"
	DefaultInstance.Region = "eu-west-1"
	DefaultInstance.Env = "lve"
	buf := bytes.NewBufferString(fmt.Sprintf(`{"credentials": "%s"}`, base64.StdEncoding.EncodeToString(encrypted)))
	Load(buf)

	config, err := AtPath("credentials").Decrypt()
	require.NoError(t, err)

	secret := config.AtPath("secret").AsString("")
	assert.Equal(t, "supersecretstring", secret)

	// Decrypt again to ensure the cache is working
	config, err = AtPath("credentials").Decrypt()
	require.NoError(t, err)

	secret = config.AtPath("secret").AsString("")
	assert.Equal(t, "supersecretstring", secret)
}
