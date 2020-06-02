package encryption

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kms"
)

func TestEncryptDecrypt(t *testing.T) {
	val := "supersecretstring"

	kmsApi := &MockKMSAPI{}
	kmsApi.On("GenerateDataKey", &kms.GenerateDataKeyInput{
		EncryptionContext: map[string]*string{"service": aws.String("test")},
		KeySpec:           aws.String("AES_256"),
		KeyId:             aws.String("master-key-id"),
	}).Return(&kms.GenerateDataKeyOutput{
		KeyId:          aws.String("key-id"),
		CiphertextBlob: []byte("ciphertext-blob"),
		Plaintext:      make([]byte, 32),
	}, nil)
	kmsApi.On("Decrypt", &kms.DecryptInput{
		EncryptionContext: map[string]*string{"service": aws.String("test")},
		CiphertextBlob:    []byte("ciphertext-blob"),
	}).Return(&kms.DecryptOutput{
		KeyId:     aws.String("key-id"),
		Plaintext: make([]byte, 32),
	}, nil)

	encryptor := KMSEncryptor{KMS: kmsApi}

	ctx := map[string]string{"service": "test"}

	encrypted, err := encryptor.Encrypt("master-key-id", ctx, []byte(val))
	require.Nil(t, err)

	decrypted, err := encryptor.Decrypt(ctx, encrypted)
	require.Nil(t, err)

	assert.Equal(t, val, string(decrypted))
}

func TestDecryptInvalidKMSCiphertext(t *testing.T) {
	val := "supersecretstring"

	kmsApi := &MockKMSAPI{}
	kmsApi.On("GenerateDataKey", &kms.GenerateDataKeyInput{
		EncryptionContext: map[string]*string{"service": aws.String("test")},
		KeySpec:           aws.String("AES_256"),
		KeyId:             aws.String("master-key-id"),
	}).Return(&kms.GenerateDataKeyOutput{
		KeyId:          aws.String("key-id"),
		CiphertextBlob: []byte("ciphertext-blob"),
		Plaintext:      make([]byte, 32),
	}, nil)
	kmsApi.On("Decrypt", mock.AnythingOfType("*kms.DecryptInput")).Return(
		nil, awserr.New("InvalidCiphertextException", "", nil),
	)

	encryptor := KMSEncryptor{KMS: kmsApi}

	ctx := map[string]string{"service": "test"}

	encrypted, err := encryptor.Encrypt("master-key-id", ctx, []byte(val))
	require.Nil(t, err)

	_, err = encryptor.Decrypt(ctx, encrypted)
	require.NotNil(t, err)
	assert.Equal(t, "unable to decrypt data key", err.Error())
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	kmsApi := &MockKMSAPI{}
	kmsApi.On("Decrypt", mock.AnythingOfType("*kms.DecryptInput")).Return(&kms.DecryptOutput{
		KeyId:     aws.String("key-id"),
		Plaintext: make([]byte, 32),
	}, nil)

	encryptor := KMSEncryptor{KMS: kmsApi}

	ctx := map[string]string{"service": "test"}

	encrypted := join([]byte("badciphertext"), make([]byte, 32))

	_, err := encryptor.Decrypt(ctx, encrypted)
	require.NotNil(t, err)
	assert.Equal(t, "cipher: message authentication failed", err.Error())
}
