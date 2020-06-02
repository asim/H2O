// Functions for generating certificates during testing (this is necessary because the Thin API <--> H1 link is
// encrypted)

package handler

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"time"

	log "github.com/cihub/seelog"
)

func generateCertificateKeyPair(validFor time.Duration, hosts []string) (cert, key io.ReadWriter, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Errorf("[Certificate generator] Failed to generate private key: %s", err.Error())
		return nil, nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Errorf("[Certificate generator] Failed to generate serial number: %s", err.Error())
		return nil, nil, err
	}

	// Separate hosts out into DNS names, and IP addresses
	dnsNames := []string{}
	ips := []net.IP{}
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			ips = append(ips, ip)
		} else {
			dnsNames = append(dnsNames, host)
		}
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Hailo"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ips,
		IsCA:                  true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Errorf("[Certificate generator] Failed to create certificate: %s", err.Error())
		return nil, nil, err
	}

	certOut := bytes.NewBuffer([]byte{})
	pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	log.Trace("[Certificate generator] Written certificate")

	keyOut := bytes.NewBuffer([]byte{})
	pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	log.Trace("[Certificate generator] Written private key")

	return certOut, keyOut, nil
}
