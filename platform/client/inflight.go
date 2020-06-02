package client

import (
	"sync"
)

// Inflight contains a list of responses which we are currently awaiting responses
type inflight struct {
	sync.RWMutex
	m map[string]chan *Response
}

func (self *inflight) add(req *Request, ch chan *Response) {
	self.Lock()
	defer self.Unlock()
	self.m[req.messageID] = ch
}

func (self *inflight) removeByRequest(req *Request) {
	self.remove(req.MessageID())
}

func (self *inflight) removeByResponse(rsp *Response) {
	self.remove(rsp.CorrelationID())
}

func (self *inflight) remove(id string) {
	self.Lock()
	defer self.Unlock()

	if ch, ok := self.m[id]; ok {
		delete(self.m, id)
		close(ch)
	}
}

// more of a getAndRemove
func (self *inflight) get(rsp *Response) (ch chan *Response, ok bool) {
	self.Lock()
	defer self.Unlock()
	ch, ok = self.m[rsp.CorrelationID()]
	if ok {
		delete(self.m, rsp.CorrelationID())
	}
	return
}
