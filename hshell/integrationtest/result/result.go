package result

import (
	"fmt"
	//"sync"
	"time"
)

var Rchan chan *RequestResult = make(chan *RequestResult)
var getChan chan map[string]*AggregateResult = make(chan map[string]*AggregateResult)
var reqChan chan bool = make(chan bool)

var ResultMap map[string]*AggregateResult = make(map[string]*AggregateResult)

//var Lock sync.RWMutex

type AggregateResult struct {
	Errors   int
	Success  int
	TotalDur time.Duration
}

type RequestResult struct {
	Name           string
	Success        bool
	Dur            time.Duration
	Response       string
	Request        interface{}
	RequestPayload string
}

func init() {
	fmt.Println("Starting result recorder")
	go Init()
}

func Init() {
	var r *RequestResult
	for {
		select {
		case r = <-Rchan:
			ar, ok := ResultMap[r.Name]
			if !ok {
				ar = &AggregateResult{}
			}
			if r.Success {
				ar.Success++
			} else {
				ar.Errors++
			}
			ar.TotalDur += r.Dur
			ResultMap[r.Name] = ar
			fmt.Printf("%+v :: %+v :: %+v :: %+v\n", r.Name, r.Request, r.RequestPayload, r.Response)
		case <-reqChan:
			getChan <- ResultMap
		}

	}
}

func GetResults() map[string]*AggregateResult {
	reqChan <- true
	return <-getChan

}
