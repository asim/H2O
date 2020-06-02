// Trace package facilitates the recording of trace events which allow us to follow
// a request right the way through the H2 platform
package trace

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/config"
	"io"
	"net"
	"sync"
	"time"
)

const (
	// Decreased from g2s limit of 64KB which seems a bit optimistic
	// If this size is a problem it could probably be increased
	// from 512B as its being sent to localhost
	MAX_PACKET_SIZE = 512 - 8 - 20 // 8-byte UDP header, 20-byte IP header
)

var (
	// Our Tracer client
	defaultClient *Tracer

	// The trace channel is used to pass traces to a goroutine that sends them to Phosphor
	// We use this with a buffered channel so that any slowness causes traces to be dropped,
	// rather than slowing down requests, and also so we don't fire off a million goroutines
	traceChan chan []byte
)

func init() {
	// Fire up the trace channel with some capacity
	traceChan = make(chan []byte, 200)

	// Initialise a Tracer client, and run its publisher
	defaultClient = New()
	go defaultClient.publisher()
}

type Tracer struct {
	sync.RWMutex
	confHash string
	phosphor *Phosphor
}

// New will mint a new Tracer - getting phosphor connection details
// from the config service and then looking out for any changes.
func New() *Tracer {
	ch := config.SubscribeChanges()
	addr := loadPhosphorAddr()

	tr := &Tracer{
		confHash: addr,
		phosphor: loadPhosphor(addr),
	}

	// launch listener for config changes
	go func() {
		for _ = range ch {
			addr := loadPhosphorAddr()
			tr.Lock()
			if addr != tr.confHash {
				tr.phosphor.Close()
				tr.phosphor = loadPhosphor(addr)
			}
			tr.Unlock()
		}
	}()

	return tr
}

func loadPhosphorAddr() string {
	host := config.AtPath("hailo", "service", "trace", "phosphor", "host").AsString("localhost")
	port := config.AtPath("hailo", "service", "trace", "phosphor", "port").AsInt(8130)
	return fmt.Sprintf("%s:%v", host, port)
}

func loadPhosphor(addr string) *Phosphor {
	p, err := dial("udp", addr)
	if err != nil {
		log.Warnf("Error initialising phosphor connection to %v", addr)
		return nil
	}

	return p
}

// publisher waits for traces sent on a channel and publishes them to the client
func (t *Tracer) publisher() {

	// Publish traces sent on the channel
	for {
		msg := <-traceChan
		t.RLock()
		if t.phosphor != nil {
			t.phosphor.Publish(msg)
		}
		t.RUnlock()
	}
}

// A Phosphor client is just something which has an io.Writer
type Phosphor struct {
	w io.Writer
}

func dial(protocol, addr string) (*Phosphor, error) {
	c, err := net.DialTimeout(protocol, addr, 2*time.Second)
	if err != nil {
		return nil, err
	}
	return newPhosphor(c)
}

func newPhosphor(w io.Writer) (*Phosphor, error) {
	return &Phosphor{
		w: w,
	}, nil
}

// Write trace to the connection as packets
// Make sure these are smaller than MAX_PACKET_SIZE
// We then write them into the io.Writer
func (p *Phosphor) Publish(msg []byte) {
	// In the base case, when the Phosphor struct is backed by a net.Conn,
	// "Multiple goroutines may invoke methods on a Conn simultaneously."
	//   -- http://golang.org/pkg/net/#Conn
	// Otherwise, Bring Your Own Synchronization™.

	_, err := p.w.Write(msg)
	if err != nil {
		log.Debugf("[Phosphor] Publish error: %s", err)
	}
}

// Close the UDP socket
func (p *Phosphor) Close() error {
	return p.w.(net.Conn).Close()
}
