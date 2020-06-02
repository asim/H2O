package hipchat

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/HailoOSS/service/config"
	"github.com/mreiferson/go-httpclient"
)

const apiUrl = "https://api.hipchat.com/v1/rooms/message"

var defaultNotifier = NewHttpNotifier()

// HttpNotifier is a notifier that talks to the Hipchat API via HTTP
type HttpNotifier struct {
	transport *httpclient.Transport
	client    *http.Client
	authToken string
}

// NewHttpNotifier mints an HTTP notifier, configuring it from the config service
func NewHttpNotifier() *HttpNotifier {
	n := &HttpNotifier{}

	n.transport = &httpclient.Transport{}
	n.transport.ConnectTimeout = config.AtPath("hailo", "service", "hipchat", "connectTimeout").AsDuration("1s")
	n.transport.RequestTimeout = config.AtPath("hailo", "service", "hipchat", "requestTimeout").AsDuration("10s")
	n.transport.ResponseHeaderTimeout = config.AtPath("hailo", "service", "hipchat", "responseHeaderTimeout").AsDuration("5s")

	n.authToken = config.AtPath("hailo", "service", "hipchat", "authToken").AsString("811e7e99296f4b2310b3e0a9bc6206")

	n.client = &http.Client{Transport: n.transport}

	return n
}

// Notify sends a message
func (n *HttpNotifier) Notify(msg *Message) error {
	if err := msg.Valid(); err != nil {
		return err
	}

	data := url.Values{}
	data.Add("room_id", msg.Room)
	data.Add("from", msg.From)
	data.Add("message", msg.Body)
	data.Add("message_format", string(msg.Format))
	if msg.Notify {
		data.Add("notify", "1")
	}
	data.Add("color", string(msg.Colour))
	data.Add("format", "json")
	data.Add("auth_token", n.authToken)

	rsp, err := n.client.PostForm(apiUrl, data)
	if err != nil {
		return fmt.Errorf("HTTP POST failed: %v", err)
	}
	defer rsp.Body.Close()

	return nil
}

// Notify wraps DefaultNotifier.Notify
func Notify(msg *Message) error {
	return defaultNotifier.Notify(msg)
}
