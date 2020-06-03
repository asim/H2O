package domain

import (
	instances "github.com/hailo-platform/H2O/discovery-service/proto/instances"
	serviceup "github.com/hailo-platform/H2O/discovery-service/proto/serviceup"
	"github.com/hailo-platform/H2O/platform/raven"
	"strconv"
)

type RabbitExchange struct {
	Hostname string // hostname of broker it lives on
	Hostport int
	Name     string // name of exchange
	Xtype    string
	Options  map[string]interface{}
}

type RabbitQueue struct {
	Hostname string
	Hostport int
	Name     string
	Options  map[string]interface{}
}

type ExchangeBinding struct {
	FromExchange string
	ToExchange   string
}

// Json definition of a binding for communicating with rabbit admin
type BindingDef struct {
	Source          string                 `json:"source"`
	Vhost           string                 `json:"vhost"`
	Destination     string                 `json:"destination"`
	DestinationType string                 `json:"destination_type"`
	RoutingKey      string                 `json:"routing_key"`
	PropertiesKey   string                 `json:"properties_key,omitempty"`
	Arguments       map[string]interface{} `json:"arguments,omitempty"`
}

func (this *BindingDef) Equals(other *BindingDef) bool {
	if this.Source != other.Source {
		return false
	}
	if this.Vhost != other.Vhost {
		return false
	}
	if this.Destination != other.Destination {
		return false
	}
	if this.RoutingKey != other.RoutingKey {
		return false
	}
	if this.Arguments != nil && other.Arguments == nil {
		return false
	}
	if this.Arguments == nil && other.Arguments != nil {
		return false
	}

	// omit properties key
	if this.Arguments != nil && other.Arguments != nil {
		if len(this.Arguments) != len(other.Arguments) {
			return false
		}
		for k, v := range this.Arguments {
			if v != other.Arguments[k] {
				return false
			}
		}
	}
	return true
}

// Json definition of an exchange for communicating with rabbit admin
type ExchangeDef struct {
	Name       string                 `json:"name"`
	Vhost      string                 `json:"vhost"`
	Type       string                 `json:"type"`
	Durable    bool                   `json:"durable"`
	AutoDelete bool                   `json:"auto_delete"`
	Internal   bool                   `json:"internal"`
	Arguments  map[string]interface{} `json:"arguments"`
	Policy     string                 `json:"policy"`
}

type RabbitHost struct {
	Host   string
	AzName string
}

// A Rule defines how a service should be bound
type Rule struct {
	Service string
	Version string // technically this is a number but string is more flexible and comparison operations work the same
	Weight  int32
}

func (this *Rule) IsApplicable(s *Service) bool {
	return this.Service == s.Service && this.Version == s.Version
}

func (this *Rule) GetRuleMap() map[string]interface{} {
	return map[string]interface{}{"service": this.Service, "x-weight": float64(this.Weight)} // cast to float because json numbers are floats
}

func (this *BindingDef) GetDestTypeCode() string {
	if this.DestinationType != "" && len(this.DestinationType) > 0 {
		return this.DestinationType[0:1]
	} else {
		return ""
	}

}

type Service struct {
	Service       string
	Version       string
	Instance      string
	AzName        string
	Subscriptions []string
}

type DestinationTypeL string

const (
	QUEUE    DestinationTypeL = "queue"
	EXCHANGE DestinationTypeL = "exchange"
)

type DestinationTypeS string

const (
	QUEUE_S    DestinationTypeS = "q"
	EXCHANGE_S DestinationTypeS = "e"
)

func ServiceFromInstancesProto(i *instances.Instance) *Service {
	return &Service{Service: i.GetServiceName(), Version: strconv.FormatUint(i.GetServiceVersion(), 10), Instance: i.GetInstanceId(), AzName: i.GetAzName(), Subscriptions: i.GetSubTopic()}
}

func ServiceFromServiceupProto(i *serviceup.Request) *Service {
	return &Service{Service: i.GetServiceName(), Version: strconv.FormatUint(i.GetServiceVersion(), 10), Instance: i.GetInstanceId(), AzName: i.GetAzName(), Subscriptions: i.GetSubTopic()}
}

func BindingDefFromService(s *Service) *BindingDef {
	return &BindingDef{Source: raven.EXCHANGE, Vhost: "/", Destination: s.Instance, DestinationType: string(QUEUE), RoutingKey: s.Service, Arguments: map[string]interface{}{"x-match": "all", "service": s.Service}}
}

func ExchangeBindingDefFromService(s *Service, azName string) *BindingDef {
	return &BindingDef{Source: raven.EXCHANGE, Vhost: "/", Destination: azName, DestinationType: string(EXCHANGE), RoutingKey: s.Service, Arguments: map[string]interface{}{"x-match": "all", "x-nofed": "yes", "service": s.Service}}
}
