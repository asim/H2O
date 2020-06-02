package mgr

import (
	"encoding/json"
	"fmt"
	"sync"

	log "github.com/cihub/seelog"

	"github.com/goamz/goamz/autoscaling"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/dynamodb"
	"github.com/goamz/goamz/ec2"
	"github.com/goamz/goamz/iam"
	"github.com/goamz/goamz/s3"
	"github.com/goamz/goamz/sqs"

	"github.com/HailoOSS/service/aws/sts"
	"github.com/HailoOSS/service/config"
)

// AwsMgr encapsulates an AWS Manager object
type AwsMgr struct {
	sync.RWMutex
	Accounts   []*AWSAccount
	sts        *sts.STSConnectionManager
	configPath string
}

// AWSAccount holds aws credentials and can be marshlled to/from JSON stored in the config service
// Params
// Id - the AWS Account id
// Regions - the supported regions for this account (currently we support eu-west-1, us-east-1 and ap-northeast-1). You might
// want to restrict the regions to only the local region depending on your use case
// SNSRole - the designated role for your service for this account
type AWSAccount struct {
	Id      string   `json:"id"`
	Regions []string `json:"regions"`
	SNSRole string   `json:"snsRole"`
}

func (a *AWSAccount) String() string {
	if a == nil {
		return ""
	}
	return fmt.Sprintf("%s:%s:%v", a.Id, a.SNSRole, a.Regions)
}

// newAWsMgr returns a new AWS mgr instance
// configPath determines where the awsAccounts config is stored in the config service in respect to your service
// For example hailo/service/foo/awsAccounts.
func newAwsMgr(configPath ...string) *AwsMgr {

	m := &AwsMgr{
		Accounts: loadAccConfig(configPath...),
		sts:      sts.NewSTSConnectionManager(),
	}

	ch := config.SubscribeChanges()
	hash, _ := config.LastLoaded()
	// Launch our config updater
	go func() {
		for _ = range ch {
			newHash, _ := config.LastLoaded()
			if hash != newHash {
				hash = newHash
				accs := loadAccConfig(configPath...)
				m.Lock()
				if len(accs) > 0 {
					m.Accounts = accs
					log.Debugf("[AWS Manager] Updating AWS Accounts:%v", m.Accounts)
				}
				m.Unlock()
			}
		}
	}()
	log.Debugf("[AWS Manager] Accounts: %v", m.Accounts)
	return m

}

// GetAuth returns an AWS auth object based on account id
func (m *AwsMgr) GetAuth(accId string) (*aws.Auth, error) {
	var account *AWSAccount
	for _, acc := range m.GetAWSAccounts() {
		if acc.Id == accId {
			account = acc
		}
	}
	if account == nil || account.Id == "" {
		return nil, fmt.Errorf("Unable to find sns role for account %v", accId)
	}

	auth, err := m.sts.GetRoleAuth(account.SNSRole, false)
	if err != nil {
		return nil, err
	}

	return auth, nil
}

// Returns the configured AWS accounts from the config
func (m *AwsMgr) GetAWSAccounts() []*AWSAccount {
	m.RLock()
	defer m.RUnlock()
	return m.Accounts
}

// Returns the role for a specific account
func (m *AwsMgr) GetAWSAccountRole(accId string) string {
	m.RLock()
	defer m.RUnlock()
	for _, acc := range m.Accounts {
		if acc.Id == accId {
			return acc.SNSRole
		}
	}
	return ""
}

// NewSQSClient returns a new SQS client
func (m *AwsMgr) NewSQSClient(accId, region string) (*sqs.SQS, error) {
	// Get Auth
	auth, err := m.GetAuth(accId)
	if err != nil {
		return nil, err
	}
	sqs := sqs.New(*auth, aws.Regions[region])
	return sqs, nil
}

// NewIamClient returns a new Iam client
func (m *AwsMgr) NewIamClient(accId, region string) (*iam.IAM, error) {
	// Get Auth
	auth, err := m.GetAuth(accId)
	if err != nil {
		return nil, err
	}
	iam := iam.New(*auth, aws.Regions[region])
	return iam, nil
}

// NewS3Client returns a new S3 client
func (m *AwsMgr) NewS3Client(accId, region string) (*s3.S3, error) {
	// Get Auth
	auth, err := m.GetAuth(accId)
	if err != nil {
		return nil, err
	}
	s3 := s3.New(*auth, aws.Regions[region])
	return s3, nil
}

// NewEC2Client returns a new EC2 client
func (m *AwsMgr) NewEC2Client(accId, region string) (*ec2.EC2, error) {
	// Get Auth
	auth, err := m.GetAuth(accId)
	if err != nil {
		return nil, err
	}
	ec2 := ec2.New(*auth, aws.Regions[region])
	return ec2, nil
}

// NewDynamoDB returns a new DynamoDB client
func (m *AwsMgr) NewDynamoDBClient(accId, region string) (*dynamodb.Server, error) {
	// Get Auth
	auth, err := m.GetAuth(accId)
	if err != nil {
		return nil, err
	}
	dynamoDB := &dynamodb.Server{
		Auth:   *auth,
		Region: aws.Regions[region],
	}
	return dynamoDB, nil
}

// NewASClient returns a new AutoScaling client
func (m *AwsMgr) NewASClient(accId, region string) (*autoscaling.AutoScaling, error) {
	// Get Auth
	auth, err := m.GetAuth(accId)
	if err != nil {
		return nil, err
	}
	as := autoscaling.New(*auth, aws.Regions[region])
	return as, nil
}

func loadAccConfig(path ...string) []*AWSAccount {
	bytes := config.AtPath(path...).AsJson()
	accs := make([]*AWSAccount, 0)
	err := json.Unmarshal(bytes, &accs)
	if err != nil {
		log.Warnf("[AWS Manager] Failed to unmarshal AWS credential pairs from config: %s", err)
		return nil
	}
	return accs
}
