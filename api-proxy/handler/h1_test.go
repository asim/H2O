package handler

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/HailoOSS/service/config"
)

const H1ConfigJson = `{
    "controlplane": {
        "configVersion": 10001,
        "rules": {
            "test-throttle": {
                "match": {
                    "path": "/h1-dummy-test",
                    "proportion": 1.0
                },
                "action": 1
            }
        },
        "regions": {
            "eu-west-1": {
                "id": "eu-west-1",
                "status": "ONLINE",
                "apps": {
                    "default": {
                        "api": "api-driver-london.elasticride.com"
                    }
                }
            }
        },
        "hobRegions": {
            "LON": "eu-west-1"
        }
    },
    "api": {
        "proxyMappings": {
            "testy": "%s"
        }
    }
}`

// Accepts new connections on the passed listener, passing them to the passed handler function when an inbound request
// is received
func acceptOnListener(listener net.Listener, handler *func(net.Conn)) {
	for {
		conn, err := listener.Accept()
		if err == nil {
			go (*handler)(conn)
		}
	}
}

// Sets up a new Listener that will accept connections signed with the certificate and key pair provided, and sets up
// the H1 proxy to accept certificates from this Listener
func setupTLSListener(t *testing.T, cert, key io.Reader) (listener net.Listener, err error) {
	// Build TLS config, loading the certificate into it
	certBlock, err := ioutil.ReadAll(cert)
	assert.NoError(t, err, "Error reading certificate")
	keyBlock, err := ioutil.ReadAll(key)
	assert.NoError(t, err, "Error reading private key")
	tlsCert, err := tls.X509KeyPair(certBlock, keyBlock)
	assert.NoError(t, err, "Error creating X.509 certificate")

	config := &tls.Config{
		NextProtos:   []string{"http/1.1"},
		Certificates: []tls.Certificate{tlsCert},
	}

	// Install the TLS config to the proxy
	proxyCertPool := x509.NewCertPool()
	ok := proxyCertPool.AppendCertsFromPEM(certBlock)
	assert.True(t, ok, "Failed to append certificate to proxy pool")
	proxyTransportTLSConfig = &tls.Config{
		RootCAs:   proxyCertPool,
		ClientCAs: proxyCertPool,
	}
	setupProxy()

	// The TLS listener wraps a normal Listener: create one of those (on a system-chosen random, unused port)
	rawListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	return tls.NewListener(rawListener, config), nil
}

// Test a normal proxied response makes it through correctly
func TestH1Proxy_Normal(t *testing.T) {
	cert, key, err := generateCertificateKeyPair(30*time.Minute, []string{"127.0.0.1", "testy"})
	assert.NoError(t, err, "Error generating certificate + private key pair")
	dummyH1ServerListener, err := setupTLSListener(t, cert, key)
	assert.NoError(t, err, "Error creating H1 server listener")
	defer dummyH1ServerListener.Close()

	// Setup a host happing to send traffic with the "testy" Host (which we'll set on all requests) to the dummy H1
	// listener
	configJson := fmt.Sprintf(H1ConfigJson, dummyH1ServerListener.Addr().String())
	buf := bytes.NewBufferString(configJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(buf)
	defer config.Load(origConfigBuf)

	server, client := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	// A dummy handler
	dummyH1ServerHandler := func(rawConn net.Conn) {
		defer rawConn.Close()

		reader := bufio.NewReader(rawConn)
		_, err := http.ReadRequest(reader)
		assert.NoError(t, err, "Error reading HTTP request in dummy proxy server")

		response := fmt.Sprintf("HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/plain\r\n"+
			"Content-Length: %d\r\n"+
			"X-Hailo-Is-Cruft: Almost certainly\r\n"+
			"\r\n"+
			"Foo bar baz", len([]byte("Foo bar baz")))

		rawConn.Write([]byte(response))
	}
	go acceptOnListener(dummyH1ServerListener, &dummyH1ServerHandler)

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/h1-dummy-test", server.URL), nil)
	assert.NoError(t, err, "Request construction error")
	request.Host = "testy"
	response, err := client.Do(request)
	assert.NoError(t, err, "Request error")
	responseBodyBytes, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err, "Error reading response body")
	responseBody := string(responseBodyBytes)

	assert.Equal(t, "Foo bar baz", responseBody, "Body mismatch")
	assert.Equal(t, "text/plain", response.Header.Get("Content-Type"), "Content-Type header mismatch")
	assert.Equal(t, int64(len([]byte("Foo bar baz"))), response.ContentLength, "Content-Length mismatch")
	assert.Equal(t, "Almost certainly", response.Header.Get("X-Hailo-Is-Cruft"), "Custom header mismatch")
}

// Test a response with an invalid Content-Length is relayed to the client as-is (our API should not screw with the
// responses sent to the client, just pass them right through)
func TestH1Proxy_InvalidResponseContentLength(t *testing.T) {
	cert, key, err := generateCertificateKeyPair(30*time.Minute, []string{"127.0.0.1", "testy"})
	assert.NoError(t, err, "Error generating certificate + private key pair")
	dummyH1ServerListener, err := setupTLSListener(t, cert, key)
	assert.NoError(t, err, "Error creating H1 server listener")
	defer dummyH1ServerListener.Close()

	// Setup a host happing to send traffic with the "testy" Host (which we'll set on all requests) to the dummy H1
	// listener
	configJson := fmt.Sprintf(H1ConfigJson, dummyH1ServerListener.Addr().String())
	buf := bytes.NewBufferString(configJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(buf)
	defer config.Load(origConfigBuf)

	server, client := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	// A dummy handler
	dummyH1ServerHandler := func(rawConn net.Conn) {
		defer rawConn.Close()

		reader := bufio.NewReader(rawConn)
		_, err := http.ReadRequest(reader)
		assert.NoError(t, err, "Error reading HTTP request in dummy proxy server")

		response := "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain\r\n" +
			"Content-Length: 2000\r\n" +
			"X-Hailo-Is-Cruft: Almost certainly\r\n" +
			"\r\n" +
			"Foo bar baz"

		rawConn.Write([]byte(response))
	}
	go acceptOnListener(dummyH1ServerListener, &dummyH1ServerHandler)

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/h1-dummy-test", server.URL), nil)
	assert.NoError(t, err, "Request construction error")
	request.Host = "testy"
	response, err := client.Do(request)
	assert.NoError(t, err, "Request error")

	assert.Equal(t, "text/plain", response.Header.Get("Content-Type"), "Content-Type header mismatch")
	assert.Equal(t, int64(2000), response.ContentLength, "Content-Length mismatch")
	assert.Equal(t, "Almost certainly", response.Header.Get("X-Hailo-Is-Cruft"), "Custom header mismatch")
}

// Test that a hanging response (à la reverse-Slowloris) times out before 60 seconds (the ELB timeout)
func TestH1Proxy_HangingResponse(t *testing.T) {
	cert, key, err := generateCertificateKeyPair(30*time.Minute, []string{"127.0.0.1", "testy"})
	assert.NoError(t, err, "Error generating certificate + private key pair")
	dummyH1ServerListener, err := setupTLSListener(t, cert, key)
	assert.NoError(t, err, "Error creating H1 server listener")
	defer dummyH1ServerListener.Close()

	// Setup a host happing to send traffic with the "testy" Host (which we'll set on all requests) to the dummy H1
	// listener
	configJson := fmt.Sprintf(H1ConfigJson, dummyH1ServerListener.Addr().String())
	buf := bytes.NewBufferString(configJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(buf)
	defer config.Load(origConfigBuf)

	server, client := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	// A dummy handler
	dummyH1ServerHandler := func(rawConn net.Conn) {
		defer rawConn.Close()

		reader := bufio.NewReader(rawConn)
		_, err := http.ReadRequest(reader)
		assert.NoError(t, err, "Error reading HTTP request in dummy proxy server")

		response := "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain\r\n" +
			"Content-Length: 10000\r\n" +
			"\r\n" +
			"Imma just leave this hanging here"

		rawConn.Write([]byte(response))
		time.Sleep(61 * time.Second)
	}
	go acceptOnListener(dummyH1ServerListener, &dummyH1ServerHandler)

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/h1-dummy-test", server.URL), nil)
	assert.NoError(t, err, "Request construction error")
	request.Host = "testy"
	_, err = client.Do(request)
	assert.Error(t, err, "Expected timeout error")
}

// Test a proxied response gets any CORS headers it may have had stripped out
func TestH1Proxy_StripsCORSHeaders(t *testing.T) {
	cert, key, err := generateCertificateKeyPair(30*time.Minute, []string{"127.0.0.1", "testy"})
	assert.NoError(t, err, "Error generating certificate + private key pair")
	dummyH1ServerListener, err := setupTLSListener(t, cert, key)
	assert.NoError(t, err, "Error creating H1 server listener")
	defer dummyH1ServerListener.Close()

	// Setup a host happing to send traffic with the "testy" Host (which we'll set on all requests) to the dummy H1
	// listener
	configJson := fmt.Sprintf(H1ConfigJson, dummyH1ServerListener.Addr().String())
	buf := bytes.NewBufferString(configJson)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(buf)
	defer config.Load(origConfigBuf)

	server, client := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	// A dummy handler
	dummyH1ServerHandler := func(rawConn net.Conn) {
		defer rawConn.Close()

		reader := bufio.NewReader(rawConn)
		_, err := http.ReadRequest(reader)
		assert.NoError(t, err, "Error reading HTTP request in dummy proxy server")

		response := fmt.Sprintf("HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/plain\r\n"+
			"Content-Length: %d\r\n"+
			"Access-Control-Allow-Origin: http://foo.bar\r\n"+
			"Access-Control-Allow-Methods: GET, POST\r\n"+
			"X-Hailo-Is-Cruft: Almost certainly\r\n"+
			"\r\n"+
			"Foo bar baz", len([]byte("Foo bar baz")))

		rawConn.Write([]byte(response))
	}
	go acceptOnListener(dummyH1ServerListener, &dummyH1ServerHandler)

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/h1-dummy-test", server.URL), nil)
	assert.NoError(t, err, "Request construction error")
	request.Host = "testy"
	response, err := client.Do(request)
	assert.NoError(t, err, "Request error")

	assert.Empty(t, response.Header.Get("Access-Control-Allow-Origin"), "CORS header was passed through from H1")
	assert.Empty(t, response.Header.Get("Access-Control-Allow-Methods"), "CORS header was passed through from H1")
}
