package encryption

import "github.com/stretchr/testify/mock"

import "github.com/aws/aws-sdk-go/service/kms"

type MockKMSAPI struct {
	mock.Mock
}

// GenerateDataKey provides a mock function with given fields: _a0
func (_m *MockKMSAPI) GenerateDataKey(_a0 *kms.GenerateDataKeyInput) (*kms.GenerateDataKeyOutput, error) {
	ret := _m.Called(_a0)

	var r0 *kms.GenerateDataKeyOutput
	if rf, ok := ret.Get(0).(func(*kms.GenerateDataKeyInput) *kms.GenerateDataKeyOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kms.GenerateDataKeyOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kms.GenerateDataKeyInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Decrypt provides a mock function with given fields: _a0
func (_m *MockKMSAPI) Decrypt(_a0 *kms.DecryptInput) (*kms.DecryptOutput, error) {
	ret := _m.Called(_a0)

	var r0 *kms.DecryptOutput
	if rf, ok := ret.Get(0).(func(*kms.DecryptInput) *kms.DecryptOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kms.DecryptOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kms.DecryptInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
