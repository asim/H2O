package handler

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mreiferson/go-httpclient"
	"github.com/stretchr/testify/assert"

	ptesting "github.com/hailo-platform/H2O/platform/testing"
)

type TestServer struct {
	*HailoServer
	listener net.Listener
	URL      *url.URL
	logger   log.LoggerInterface
}

func (s *TestServer) Serve() {
	s.HailoServer.HTTPServer.Serve(s.listener)
}

func SetupTestServerAndClient(t *testing.T) (server *TestServer, client *http.Client) {
	// Discard the access logs
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err, "Error listening on 127.0.0.1:0")
	t.Logf("Test server listening on %v", listener.Addr())

	serverURL, err := url.Parse(fmt.Sprintf("http://%s", listener.Addr().String()))
	assert.NoError(t, err, "Error parsing listener Addr() to URL")

	logger := ptesting.InstallTestLoggerWithLogLevel(t, "off")

	server = &TestServer{
		HailoServer: NewServer(ioutil.Discard),
		listener:    listener,
		URL:         serverURL,
		logger:      logger,
	}
	go server.Serve()

	client = &http.Client{
		Transport: &httpclient.Transport{
			ConnectTimeout:        1 * time.Second,
			RequestTimeout:        10 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
		}}
	return server, client
}

func TeardownTestServer(t *testing.T, s *TestServer) {
	s.Kill(fmt.Errorf("TeardownTestServer called"))
	s.listener.Close()
	log.ReplaceLogger(s.logger)
}

// Test that if a client connection hangs (à la Slowloris attack), it is closed before 60 seconds (the ELB timeout)
func TestRegression_HangingConnectionTimeout(t *testing.T) {
	server, _ := SetupTestServerAndClient(t)
	defer TeardownTestServer(t, server)

	conn, err := net.Dial("tcp", server.listener.Addr().String())
	assert.NoError(t, err)
	fmt.Fprint(conn, "GET / HTTP/1.1\r\n"+
		"User-Agent: Break all the things\r\n"+
		"Host: 127.0.0.1\r\n"+
		"Accept: */*\r\n"+
		"Content-Length: 1000\r\n"+
		"\r\n"+
		"foobar")

	receiver := func(c net.Conn, resultChan chan<- []byte) {
		resultBytes := make([]byte, 100)
		_, err := c.Read(resultBytes)
		if err == nil {
			resultChan <- resultBytes
		} else {
			close(resultChan)
		}
	}
	receiverChan := make(chan []byte)
	go receiver(conn, receiverChan)

	timer := time.NewTimer(60 * time.Second)
	startTime := time.Now()
	defer timer.Stop()
	select {
	case <-timer.C:
		assert.Fail(t, "Connection was not closed by the server before 60 seconds")
	case <-receiverChan:
		timeoutDuration := time.Now().Sub(startTime)
		t.Logf("Successfully timed out in %f seconds", timeoutDuration.Seconds())
		return
	}
}
