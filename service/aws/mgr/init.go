// Package mgr provides an easy way of abstracting AWS Client API auth using STS Auth
// It relies on AWS Account configuration being set up for the service using it
//
// As it is using STS by default, the auth for the clients is done via STS token and
// those tokens are cached until they expire by the STS manager. In practical terms this
// means that is it safe ( and you should ) reinitialise your AWS clients as you need them
// as opposed to holding a shared client object.

package mgr

import (
	"fmt"

	log "github.com/cihub/seelog"

	"github.com/goamz/goamz/autoscaling"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/dynamodb"
	"github.com/goamz/goamz/ec2"
	"github.com/goamz/goamz/iam"
	"github.com/goamz/goamz/s3"
	"github.com/goamz/goamz/sqs"
)

var (
	mgr *AwsMgr
)

// Init initialises the package mgr
// configPath determines where the awsAccounts config is stored in the config service in respect to your service
// For example hailo/service/foo/awsAccounts.
func Init(configPath ...string) {
	mgr = newAwsMgr(configPath...)
}

func GetAuth(accId string) (*aws.Auth, error) {
	if mgr == nil {
		return nil, fmt.Errorf("AWS Mgr package not initialised! Use the Init method first!")
	}
	return mgr.GetAuth(accId)
}

func GetAWSAccounts() []*AWSAccount {
	if mgr == nil {
		log.Errorf("AWS Mgr package not initialised! Use the Init method first!")
		return nil
	}
	return mgr.GetAWSAccounts()
}

func GetAWSAccountRole(accId string) string {
	if mgr == nil {
		log.Errorf("AWS Mgr package not initialised! Use the Init method first!")
		return ""
	}
	return mgr.GetAWSAccountRole(accId)
}

func NewSQSClient(accId, region string) (*sqs.SQS, error) {
	if mgr == nil {
		return nil, fmt.Errorf("AWS Mgr package not initialised! Use the Init method first!")
	}
	return mgr.NewSQSClient(accId, region)
}

func NewIamClient(accId, region string) (*iam.IAM, error) {
	if mgr == nil {
		return nil, fmt.Errorf("AWS Mgr package not initialised! Use the Init method first!")
	}
	return mgr.NewIamClient(accId, region)
}

func NewS3Client(accId, region string) (*s3.S3, error) {
	if mgr == nil {
		return nil, fmt.Errorf("AWS Mgr package not initialised! Use the Init method first!")
	}
	return mgr.NewS3Client(accId, region)
}

func NewEC2Client(accId, region string) (*ec2.EC2, error) {
	if mgr == nil {
		return nil, fmt.Errorf("AWS Mgr package not initialised! Use the Init method first!")
	}
	return mgr.NewEC2Client(accId, region)
}

func NewASClient(accId, region string) (*autoscaling.AutoScaling, error) {
	if mgr == nil {
		return nil, fmt.Errorf("AWS Mgr package not initialised! Use the Init method first!")
	}
	return mgr.NewASClient(accId, region)
}

func NewDynamoDBClient(accId, region string) (*dynamodb.Server, error) {
	if mgr == nil {
		return nil, fmt.Errorf("AWS Mgr package not initialised! Use the Init method first!")
	}
	return mgr.NewDynamoDBClient(accId, region)
}
