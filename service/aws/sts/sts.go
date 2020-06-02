package sts

import (
	"crypto/rand"
	"fmt"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/sts"
	"github.com/HailoOSS/platform/util"
	"regexp"
	"strings"
	"sync"
	"time"
)

type STSConnectionManager struct {
	m map[string]*aws.Auth
	sync.RWMutex
}

func NewSTSConnectionManager() *STSConnectionManager {
	return &STSConnectionManager{
		m: make(map[string]*aws.Auth),
	}
}

func (m *STSConnectionManager) Get(key string) *aws.Auth {
	m.RLock()
	defer m.RUnlock()
	if auth, ok := m.m[key]; ok {
		return auth
	}
	return nil
}

func (m *STSConnectionManager) Set(key string, auth *aws.Auth) {
	m.Lock()
	defer m.Unlock()
	m.m[key] = auth
}

func (m *STSConnectionManager) Delete(key string) {
	m.Lock()
	defer m.Unlock()
	delete(m.m, key)
}

// GetRoleAuth - Returns an (existing) authentication token for assuming an STS role
//   roleArn - the requested role ARN (example: arn:aws:iam::11111111111:role/myrole )
//   renew - explicitely request session renewing
// Returns aws.Auth object that can be used with any of the existing goamz APIs
//
func (m *STSConnectionManager) GetRoleAuth(roleArn string, renew bool) (*aws.Auth, error) {
	accountId := arnToAccId(roleArn)
	conn := fmt.Sprintf("%s%s", accountId, arnToName(roleArn))
	auth := m.Get(conn)
	// Check if we already have a connection that hasn't expired and we
	// don't want to explicitely renew it
	if auth != nil && time.Since(auth.Expiration()).Seconds() < -300 && !renew {
		return auth, nil
	}
	auth, err := AssumeRole(roleArn, generatePseudoRand(), 3600)
	if err != nil {
		return nil, err
	}
	m.Set(conn, auth)
	return auth, nil
}

// Assume role uses the current server role to call STS and assume a different role if permitted
// Params:
//   roleArn - the requested role ARN (example: arn:aws:iam::11111111111:role/myrole )
//   sessionName - a name to associate with the current session. Use the service name +
//                 unique idenfitifier preferebly.
//   duration - the duration of the session in seconds. Must be between 900 and 3600
// Returns aws.Auth object that can be used with any of the existing goamz APIs
//
// Check http://goo.gl/M6uCu5 for more information
//
func AssumeRole(roleArn string, sessionName string, duration int) (*aws.Auth, error) {
	if duration < 900 || duration > 3600 {
		return nil, fmt.Errorf("Duration out of bounds")
	}

	//Try to get our local auth
	localAuth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		return nil, err
	}

	stsClient := sts.New(localAuth, aws.Regions[util.GetAwsRegionName()])
	stsOptions := &sts.AssumeRoleParams{
		DurationSeconds: int(duration),
		RoleArn:         roleArn,
		RoleSessionName: sessionName,
	}

	//Try to assume role
	roleAuth, err := stsClient.AssumeRole(stsOptions)
	if err != nil {
		return nil, err
	}

	//Marshal the response into an aws.Auth object
	auth := aws.NewAuth(roleAuth.Credentials.AccessKeyId, roleAuth.Credentials.SecretAccessKey,
		roleAuth.Credentials.SessionToken, roleAuth.Credentials.Expiration)

	return auth, nil
}

func generatePseudoRand() string {
	alphanum := "0123456789abcdefghigklmnopqrst"
	var bytes = make([]byte, 10)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func arnToName(s string) string {
	r := regexp.MustCompile(`arn:aws:.+\/.+`)

	if r.MatchString(s) {
		parts := strings.Split(s, "/")
		rsp := parts[len(parts)-1]
		return rsp
	}

	return s
}

func arnToAccId(s string) string {
	r := regexp.MustCompile(`arn:aws:.+\/.+`)

	if r.MatchString(s) {
		parts := strings.Split(s, ":")
		if len(parts) > 4 {
			rsp := parts[4]
			return rsp
		}
	}

	return s
}
