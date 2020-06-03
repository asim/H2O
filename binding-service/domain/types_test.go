package domain

import (
	"testing"
)

func TestServiceToBindingDef(t *testing.T) {
	s := Service{
		Service:  "com.hailo-platform/H2O.service.foobar",
		Version:  "201306271500",
		Instance: "server-com.hailo-platform/H2O.service.foobar-1234567890",
		AzName:   "eu-west-1a"}
	b := BindingDefFromService(&s)
	if b.Destination != "server-com.hailo-platform/H2O.service.foobar-1234567890" {
		t.Error("Destination incorrect ", b.Destination)
	}
	if b.DestinationType != string(QUEUE) {
		t.Error("Destination type incorrect ", b.DestinationType)
	}
	if b.RoutingKey != "com.hailo-platform/H2O.service.foobar" {
		t.Error("Routing key incorrect ", b.RoutingKey)
	}
	if b.Source != "h2o" {
		t.Error("Source incorrect ", b.Source)
	}
	if b.Vhost != "/" {
		t.Error("Vhost incorrect ", b.Vhost)
	}
	if b.Arguments["x-match"] != "all" {
		t.Error("'x-match' incorrect ", b.Arguments["x-match"])
	}
	if b.Arguments["service"] != "com.hailo-platform/H2O.service.foobar" {
		t.Error("'service' incorrect ", b.Arguments["service"])
	}
}

func TestServiceToExchangeBindingDef(t *testing.T) {
	s := Service{
		Service:  "com.hailo-platform/H2O.service.foobar",
		Version:  "201306271500",
		Instance: "server-com.hailo-platform/H2O.service.foobar-1234567890",
		AzName:   "eu-west-1a"}
	b := ExchangeBindingDefFromService(&s, "eu-west-1a")
	if b.Destination != "eu-west-1a" {
		t.Error("Destination incorrect ", b.Destination)
	}
	if b.DestinationType != string(EXCHANGE) {
		t.Error("Destination type incorrect ", b.DestinationType)
	}
	if b.RoutingKey != "com.hailo-platform/H2O.service.foobar" {
		t.Error("Routing key incorrect ", b.RoutingKey)
	}
	if b.Source != "h2o" {
		t.Error("Source incorrect ", b.Source)
	}
	if b.Vhost != "/" {
		t.Error("Vhost incorrect ", b.Vhost)
	}
	if b.Arguments["x-match"] != "all" {
		t.Error("'x-match' incorrect ", b.Arguments["x-match"])
	}
	if b.Arguments["service"] != "com.hailo-platform/H2O.service.foobar" {
		t.Error("'service' incorrect ", b.Arguments["service"])
	}
	if b.Arguments["x-nofed"] != "yes" {
		t.Error("'x-nofed' incorrect ", b.Arguments["x-nofed"])
	}
}
