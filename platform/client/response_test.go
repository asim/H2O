package client

import (
	"testing"

	"github.com/streadway/amqp"
)

func TestNewResponseMints(t *testing.T) {
	delivery := &amqp.Delivery{
		CorrelationId: "123456",
	}
	rsp := newResponseFromDelivery(*delivery)
	if rsp.CorrelationID() != "123456" {
		t.Error("Incorrect correlation ID")
	}
}

func BenchmarkNewRespones(b *testing.B) {
	delivery := &amqp.Delivery{
		CorrelationId: "123456",
	}
	for i := 0; i < b.N; i++ {
		_ = newResponseFromDelivery(*delivery)
	}
}
