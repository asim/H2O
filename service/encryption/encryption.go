package encryption

type Encryptor interface {
	Encrypt(keyID string, ctx map[string]string, plaintext []byte) ([]byte, error)
	Decrypt(ctx map[string]string, ciphertext []byte) ([]byte, error)
}
