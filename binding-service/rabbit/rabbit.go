package rabbit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var RabbitPort string = "15672"
var RabbitUser string = "hailo"
var RabbitPass string = "hailo"

/*
[{
	message_stats: {
		publish_in: 5,
		publish_in_details: {
			rate: 0
		},
		publish_out: 5,
		publish_out_details: {
			rate: 0
		}
	},
	name: "",
	vhost: "/",
	type: "direct",
	durable: true,
	auto_delete: false,
	internal: false,
	arguments: {
		alternate-exchange: "h2o.deadletter"
	},
	policy: "federate-direct"
}]
*/

/*
type Response struct {
	Data struct {
		Children []struct {
			Data Item
		}
	}
}
*/
type Exchange struct {
	Message_stats struct {
		Publish_in         interface{}
		Publish_in_details struct {
			Rate interface{}
		}
		Publish_out         interface{}
		Publish_out_details struct {
			Rate interface{}
		}
	}
	Name        string
	Vhost       string
	Type        string
	Durable     bool
	Auto_delete bool
	Internal    bool
	Arguments   map[string]interface{}
	Policy      string
}

type Exchanges []*Exchange

func GetExchanges(rabbitHost string) ([]*Exchange, error) {
	rsp, err := callRabbit(rabbitHost, "exchanges", "GET", nil)
	if err != nil {
		return nil, err
	}
	exch := &Exchanges{}
	err = json.Unmarshal(rsp, exch)
	if err != nil {
		return nil, err
	}

	return []*Exchange(*exch), err
}

type Queue struct {
	Memory        int
	Message_stats struct {
		Deliver_get         int
		Deliver_get_details struct {
			Rate float64
		}
		Deliver_no_ack         int
		Deliver_no_ack_details struct {
			Rate float64
		}
		Publish         int
		Publish_details struct {
			Rate float64
		}
	}
	Messages         int
	Messages_details struct {
		Rate float64
	}
	Messages_ready         int
	Messages_ready_details struct {
		Rate float64
	}
	Messages_unacknowledged         int
	Messages_unacknowledged_details struct {
		Rate float64
	}
	Idle_since             string
	Policy                 string
	Exclusive_consumer_tag string
	Consumers              int
	Backing_queue_status   struct {
		Q1                   int
		Q2                   int
		Delta                []interface{}
		Q3                   int
		Q4                   int
		Len                  int
		Pending_acks         int
		Target_ram_count     string
		Ram_msg_count        int
		Ram_ack_count        int
		Next_seq_id          int
		Persistent_count     int
		Avg_ingress_rate     float64
		Avg_egress_rate      float64
		Avg_ack_ingress_rate float64
		Avg_ack_egress_rate  float64
	}
	Status      string
	Name        string
	Vhost       string
	Durable     bool
	Auto_delete bool
	Arguments   map[string]interface{}
	Node        string
}

type Queues []*Queue

func GetQueues(rabbitHost string) ([]*Queue, error) {
	rsp, err := callRabbit(rabbitHost, "queues", "GET", nil)
	if err != nil {
		return nil, err
	}
	exch := &Queues{}
	err = json.Unmarshal(rsp, exch)
	if err != nil {
		return nil, err
	}

	return []*Queue(*exch), err
}

type Binding struct {
	Source           string
	Vhost            string
	Destination      string
	Destination_type string
	Routing_key      string
	Arguments        map[string]interface{}
	Properties_key   string
}

type Bindings []*Binding

func GetBindings(rabbitHost string) ([]*Binding, error) {
	rsp, err := callRabbit(rabbitHost, "bindings", "GET", nil)
	if err != nil {
		return nil, err
	}
	exch := &Bindings{}
	err = json.Unmarshal(rsp, exch)
	if err != nil {
		return nil, err
	}

	return []*Binding(*exch), err
}

func GetQueue(rabbitHost, queue string) (*Queue, error) {
	rsp, err := callRabbit(rabbitHost, `queues/%2F/`+queue, "GET", nil)
	if err != nil {
		return nil, err
	}
	exch := &Queue{}
	err = json.Unmarshal(rsp, exch)
	if err != nil {
		fmt.Println(string(rsp))
		return nil, err
	}

	return exch, err
}

type Message struct {
	Payload_bytes int
	Redelivered   bool
	Exchange      string
	Routing_key   string
	Message_count int
	Properties    struct {
		Message_id    string
		Reply_to      string
		Delivery_mode int
		Headers       map[string]string
		Content_type  string
	}
	Payload          string
	Payload_encoding string
}

type Messages []*Message

func DeleteQueueMessages(rabbitHost, queue string) error {
	_, err := callRabbit(rabbitHost, `queues/%2F/`+queue+`/contents`, "DELETE", nil)
	if err != nil {
		fmt.Println("error deleting")
		return err
	}
	return nil
}

func GetExampleMessages(rabbitHost, queue string) ([]*Message, error) {

	jsonStr := `{"count":5,"requeue":true,"encoding":"auto","truncate":50000}`
	rsp, err := callRabbit(rabbitHost, `queues/%2F/`+queue+`/get`, "POST", []byte(jsonStr))
	if err != nil {
		fmt.Println("error posting")
		return nil, err
	}

	exch := &Messages{}

	err = json.Unmarshal(rsp, exch)
	if err != nil {
		fmt.Println(string(rsp))
		return nil, err
	}

	return []*Message(*exch), err
}

func callRabbit(rabbitHost, path, method string, b []byte) ([]byte, error) {

	req, err := http.NewRequest(method, fmt.Sprintf("%s:%s/api/%s", rabbitHost, RabbitPort, path), bytes.NewReader(b))

	if err != nil {
		return nil, err
	}
	//have to do this cruft to get away with %2F in the url path
	req.URL.Opaque = "//" + req.URL.Host + "/api/" + path

	req.SetBasicAuth(RabbitUser, RabbitPass)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	rb, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return rb, err
}
