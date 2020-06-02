package signer

import (
	"crypto"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/HailoOSS/login-service/domain"
)

const testPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIDANBgkqhkiG9w0BAQEFAAOCAQ0AMIIBCAKCAQEAxEiAaFDgroIvK6U+d4GO
Rt6s9rHpDiZ/Mpf/IcaPDHKLrnFb+HrZTVM/AAkKG6AnqejBaJMUTUgnRqq2Zjzl
sodS668L3GBxv2IJfFrtX/bAMN43zonHthJlnTredPdfmtNS0B6QFyA32Y9VLdMP
9Nbum4KHUZJK86mpoTqhLBAWFXC3uWXQD98DItWkYZQ8AgM9f9/XFOUzKg+pMG+Q
C1bUPZg0oARfpMpZGw3ksQTmDj47pl6W/NhllnSeULjHWg23LCE6XL4I9cKgkIOH
hoOgxx4BTDTtm+t2C4Kq6H52buZFUbhuxi3/Vqw9OGFYZqQ/Dd9D3OJO3BA3LoT2
mQIBIw==
-----END PUBLIC KEY-----`

const testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAxEiAaFDgroIvK6U+d4GORt6s9rHpDiZ/Mpf/IcaPDHKLrnFb
+HrZTVM/AAkKG6AnqejBaJMUTUgnRqq2ZjzlsodS668L3GBxv2IJfFrtX/bAMN43
zonHthJlnTredPdfmtNS0B6QFyA32Y9VLdMP9Nbum4KHUZJK86mpoTqhLBAWFXC3
uWXQD98DItWkYZQ8AgM9f9/XFOUzKg+pMG+QC1bUPZg0oARfpMpZGw3ksQTmDj47
pl6W/NhllnSeULjHWg23LCE6XL4I9cKgkIOHhoOgxx4BTDTtm+t2C4Kq6H52buZF
Ubhuxi3/Vqw9OGFYZqQ/Dd9D3OJO3BA3LoT2mQIBIwKCAQA9sGLe9NjmY23pJU4l
jx4WRfvYgQ67Te12TQcKnX1qT+LBy9sTlFLsap7Ffy8P/yJomZvfCadLfROofsuc
eYoE6LOpKGLQO5F2qch3ikqagMAPW8hle8JsbC6QgDdJVQ9rLHkc1mfMwPucJbuv
UPZi5HbgatoSUoyVuPrMRaBeTaWvwWGI53lRE+6YDNpefoem+N2Jr2jBYFxe4MLT
sBAjEuGATtfW2CuRvGM+roc6Fuunpt0c+oaHfGWJfZ79ng5oy+I2+pGM4CRyTMKs
y8zSRpdM76Fxmt1p4R+clpaA1Y/emRYVf6vTz+GDplUOADyInSv4Ga//7uerRViT
LE2LAoGBAO2T0AaD/mT3ag/hEd/s84ukMJfB3ZEXUFaMP8Cg4YyCR93naraUvOUx
bsKAewUxKvq5oSrGe2AVerByu2/wmtvFjlR7iv0diapUr6AQHnWyrneZKxbRIH2r
Gr9aIw8GCbcRxvtUrYA9JkvZHU+7gIn/6gF1iDdwoEpbMtvb9rVBAoGBANOA85Ik
tXS5ZBNd9/ZfPKOno4fCGfitkM37kGViXdAwu50ySeaOOpK67OZ2CncM1s75W5cm
l4rbLv20EjR++Hfp9qowMyhQazB7Nglx3J1lUaTcD7pAJ69xYYCn3Y53EwzerI9b
i/pbckMJN97b91OcxiYOkPOySd2OdRLypDNZAoGBANk2r5DttWOgYPiTQ4rnRRH1
M73zE7fazRvt8SUlYIB3HSKnsgYEVO7OHB+LaSlRhmGTxo2CROoiREmNeCvUqteQ
DRoKjbQ4Q1nnBvi+R7wRFIqaqw2MAHLllMw1J14xZ/fVvT2PP4swmArVIh0DM7Ff
AdVyxa8H4wInfvTmXeeLAoGBAKMo9mlsxn6dpPj/WOKhPWhOHxD00jTdqjiAPDD0
HHwIVjdv792+LS9c/+T7+XHIE2x3OAbjQbRCrzi+HKwnbzfvABzjWqoSJs2g4Irx
b6yenhF2j8oxfbM6NUX91sz8xY2V1ZMpXVq7kqi99+ZvLIJUXlffWeCQ2eVt5Ukh
lJyjAoGBAJHK8T5tWZlbvzcZbjIxGzgFsVynPmBezwFfmRRX1bYSwdZpTvQMYNLA
Odj06M5jnTSv/zH4IS+bkwN5Srdav7Rsp0V0zi53lxFgYmSIbxCWzKAKYqTLb+Dd
0xGcCz5dLnayLNq1VVomUydu+kZ7A/2PBCI/0BH8mGhnzOKkm7x8
-----END RSA PRIVATE KEY-----`

func mintToken() domain.Token {
	return domain.Token{
		Created:       time.Unix(1378377733, 0),
		AuthMechanism: "admin",
		DeviceType:    "cli",
		Id:            "dave",
		Expires:       time.Unix(1378406533, 0),
		AutoRenew:     time.Time{},
		Roles:         []string{"ADMIN"},
		Signature:     `IFvyIAtFhMu9Gkh/cf/2JtwZe8+Kk2jf4xSkAHuaJwtmlnvrIBAVmAPytzKv2hFumd7gyP/lPrGXutKAeBmUrWYd/NeewKRdT/vaAmNgtT9qi3QqeiloOfyieZjcruFnzMRc6XY4CJYHbXth2qUyF+gYFT0MIV3Dv6OUDNDOMymOuQ8z67rDMsyoKKKko5UY8oY+kURnBH0tkGy5IFLY0sH9LWldSJYXTzKWSQl3siwkCoCeCpOn9j4q2fx4vAitJ1+Vz078V1BgHDWkyx3AlM9QefRcypBOFZZ2x+fwSzNEqm37FU5DJyDA3vhCQn2Wn2DodJVr9NWRBSXPZ1/0zQ==`,
	}
}

func makeSigner() *defaultSigner {
	pub, _ := bytesToPublicKey([]byte(testPublicKey))
	prv, _ := bytesToPrivateKey([]byte(testPrivateKey))
	s := &defaultSigner{
		hash:       crypto.SHA1,
		pub:        pub,
		prv:        prv,
		loadedChan: make(chan struct{}),
	}
	close(s.loadedChan)
	s.loadedChanClosed = true
	return s
}

func TestVerify(t *testing.T) {
	s := makeSigner()
	token := mintToken()

	if !s.Verify(&token) {
		t.Fatalf("Failed to validate")
	}
}

func TestVerifyFails(t *testing.T) {
	s := makeSigner()
	token := mintToken()
	token.Id = "Dave"

	if s.Verify(&token) {
		t.Fatalf("Not expecting this to validate")
	}
}

func TestSign(t *testing.T) {
	s := makeSigner()
	token := mintToken()
	token.Id = "mynameis"

	// shouldn't validate
	if s.Verify(&token) {
		t.Fatalf("Not expecting this to validate since we changed ID")
	}

	// now sign it
	tNew, err := s.Sign(&token)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// should now validate
	if !s.Verify(tNew) {
		t.Fatalf("Failed to validate our new signed token")
	}
}

func TestPlatOneThreeThree(t *testing.T) {
	// am=h2.PASSENGER:d=cli:id=290840068858810368:ct=1394914340:et=1394943140:rt=1394941340:r=CUSTOMER,H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150,H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING,H4BADMIN.5a706c69-a33b-4e8f-ba40-09ee760385e3.PENDING,H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING:sig=ORh06xWVqJJhKNCE0R5I0A+Dy1vnvlqaE0bbJWwUEvvsXyWs2wLZ8UpDHmTZkYLvyjcuh5aRkXAGh5Hm7HPQG1FrnNf741fYCSONeLW+zfbQnyYwuB54L81DkL2PqD7nu6xHkbBQqz4Ja4lSzpS7/E9oW8fWQxF26SyVBTy6wkv0inrSojA86vX6dYCNqT1pt5oBG+Re5jnBYiAesZ0h7elsv9yhLw1o1w9UzZ106qcIA2PRycD3ERDFcNGaPUkYvYAQS3sZXeQ9CqgyEOvpMArrvo+AjuO/l2gbrjEC5PzebZi2CKopUIOD64IfoeDryORotnj2uFE6lYlfdUY+3Q==
	s := makeSigner()
	tok := &domain.Token{
		Created:       time.Unix(1394914340, 0),
		AuthMechanism: "h2.PASSENGER",
		DeviceType:    "cli",
		Id:            "290840068858810368",
		Expires:       time.Unix(1394943140, 0),
		AutoRenew:     time.Unix(1394941340, 0),
		Roles: []string{
			"CUSTOMER",
			"H4BADMIN.0641085f-ee14-49de-b8e2-e330e4d76150",
			"H4BADMIN.40ebd8c6-698e-4347-95af-1430ab2fb331.PENDING",
			"H4BADMIN.5a706c69-a33b-4e8f-ba40-09ee760385e3.PENDING",
			"H4BADMIN.7d793409-19b6-43d8-a165-5cb94397ab01.PENDING",
		},
	}
	signed, err := s.Sign(tok)
	if err != nil {
		t.Errorf("Failed to sign token: %v", err)
	}

	// now verify -- should verify!
	if !s.Verify(signed) {
		t.Error("Failed to verify signed token")
	}
}

func TestWaitsForKeysLoaded(t *testing.T) {
	s := &defaultSigner{
		hash:       crypto.SHA1,
		loadedChan: make(chan struct{}),
	}
	token := mintToken()

	mtx := sync.Mutex{}
	signerHasResult := false

	go func() {
		s.Verify(&token)
		_, err := s.Sign(&token)
		assert.NoError(t, err, "Error signing after keys are available")
		mtx.Lock()
		signerHasResult = true
		mtx.Unlock()
	}()

	time.Sleep(1 * time.Second)
	mtx.Lock()
	assert.False(t, signerHasResult, "Signer has result prematurely (before keys loaded)")
	mtx.Unlock()

	s.pub, _ = bytesToPublicKey([]byte(testPublicKey))
	s.prv, _ = bytesToPrivateKey([]byte(testPrivateKey))
	close(s.loadedChan)
	s.loadedChanClosed = true

	time.Sleep(1 * time.Second)
	mtx.Lock()
	assert.True(t, signerHasResult, "Signer did not return when the keys were loaded")
	mtx.Unlock()
}
