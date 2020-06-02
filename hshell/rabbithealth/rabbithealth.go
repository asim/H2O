package main

import (
	"flag"
	"fmt"
	seelog "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/raven"
	"github.com/HailoOSS/hshell/discovery"
	"github.com/HailoOSS/hshell/rabbit"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

func main() {
	seelog.ReplaceLogger(seelog.Disabled)
	log.SetOutput(ioutil.Discard)
	deadlet := flag.Bool("deadletter", true, "check the deadletter queue?")
	flag.Parse()
	hostMap := make(map[string]string)
	hostMap["10.2.2.50"] = "eu-west-1a"
	hostMap["10.2.2.49"] = "eu-west-1a"
	hostMap["10.2.2.219"] = "eu-west-1b"
	hostMap["10.2.2.218"] = "eu-west-1b"
	hostMap["10.2.3.109"] = "eu-west-1c"
	hostMap["10.2.3.108"] = "eu-west-1c"

	for rabHost, azname := range hostMap {
		fmt.Printf(addBold(fmt.Sprintf("\n#################################\nChecking AZ %s on rabbit %s\n", azname, rabHost)))
		raven.AmqpUri = fmt.Sprintf("amqp://hailo:hailo@%v:%v", rabHost, "5672")
		<-raven.Connect()

		qu, _ := rabbit.GetQueues("http://" + rabHost)
		bin, _ := rabbit.GetBindings("http://" + rabHost)
		//ex, _ := rabbit.GetExchanges("http://10.2.2.50")

		discQu, _ := discovery.CallInstances(azname)
		isBad := false
		serverQCount := 0
		clientQCount := 0
		for _, discVal := range discQu {
			inst := discVal.GetInstanceId()

			//checking queues
			qfound := false
			serverQCount = 0
			clientQCount = 0
			for _, val := range qu {
				if val.Name == inst {
					qfound = true
				}
				if strings.Contains(val.Name, "server-") {
					serverQCount++
				}

				if strings.Contains(val.Name, "client-") {
					clientQCount++
				}
			}

			if !qfound {
				isBad = true
				fmt.Printf(addRed(fmt.Sprintf("Bad queue %s\n", discVal.GetServiceName())))
				fmt.Printf("\tCould not find queue %s\n", discVal.GetInstanceId())
			}

			//checking bindings
			lbfound := false
			for _, val := range bin {
				if val.Source == "h2o" {
					if val.Routing_key == discVal.GetServiceName() {
						if val.Destination == inst {
							lbfound = true
						}
					}
				}
			}
			if !lbfound {
				isBad = true
				fmt.Printf(addRed(fmt.Sprintf("No local binding for %s\n", discVal.GetServiceName())))

			}

		}

		if serverQCount > len(discQu) {
			isBad = true
			fmt.Println(addRed(fmt.Sprintf("Server queue count (%v) does not match discovered services (%v)", serverQCount, len(discQu))))
			fmt.Printf("\tExcess Queues include:\n")

			for _, val := range qu {
				if strings.Contains(val.Name, "server-") {
					qfound := false
					for _, discVal := range discQu {
						if val.Name == discVal.GetInstanceId() {
							qfound = true
						}
					}
					if !qfound {
						fmt.Printf("\t\t%s\n", val.Name)
					}
				}
			}
		}
		if serverQCount < len(discQu) {
			fmt.Println(addRed(fmt.Sprintf("Server queue count (%v) does not match discovered services (%v)", serverQCount, len(discQu))))
		}

		/*if clientQCount > serverQCount {
			isBad = true
			fmt.Println(addRed(fmt.Sprintf("Client queue count (%v) is greater than server queue count (%v)", clientQCount, serverQCount)))
		}*/

		if !isBad {
			fmt.Println(addGreen(fmt.Sprintf("Bindings and Queues good (%v,%v)", serverQCount, len(discQu))))
		}
		/*for _, val := range qu {
			if val.Name == "h2odeadletter" {
				if val.Message_stats.Publish_details.Rate != 0 {
					fmt.Printf("%s %v\n", addRed("DEAD LETTER QUEUE RATE:"), val.Message_stats.Publish_details.Rate)
				} else {
					fmt.Printf("%s %v\n", addGreen("DEAD LETTER QUEUE RATE:"), val.Message_stats.Publish_details.Rate)
				}
			}
		}*/

		if *deadlet {
			rabbit.DeleteQueueMessages("http://"+rabHost, "h2odeadletter")
			deadLetVal := 0.0
			pollingMessage := "Polling Deadletter Queue"
			for ii := 0; ii < 5; ii++ {
				deadlet, err := rabbit.GetQueue("http://"+rabHost, "h2odeadletter")
				if err != nil {
					fmt.Println(err)
					break
				}
				deadLetVal += deadlet.Message_stats.Publish_details.Rate
				pollingMessage += " ."
				fmt.Println(pollingMessage)
				time.Sleep(3 * time.Second)
			}

			avDeadLet := deadLetVal / 5.0

			if avDeadLet > 0.0001 {
				fmt.Printf("%s %v\n", addRed("DEAD LETTER QUEUE RATE:"), avDeadLet)
				msg, err := rabbit.GetExampleMessages("http://"+rabHost, "h2odeadletter")
				if err != nil {
					fmt.Println(err)
				}
				for _, example := range msg {
					fmt.Printf("\tMessage on exchange %s to %s.%s \n\t\tfrom %s of type %s replying to %s\n\t\ton routing key %s\n",
						example.Exchange,
						example.Properties.Headers["service"],
						example.Properties.Headers["endpoint"],
						example.Properties.Headers["from"],
						example.Properties.Headers["messageType"],
						example.Properties.Reply_to,
						example.Routing_key,
					)
				}

			} else {
				fmt.Printf("%s %v\n", addGreen("DEAD LETTER QUEUE RATE:"), avDeadLet)
			}
		}
		if len(qu) > (len(discQu)*2)+100 {
			fmt.Println(addRed(fmt.Sprintf("Seems like a few to many queues: %v", len(qu))))
		}

		fmt.Println(addBold("#################################"))
	}
	/*bin, err := rabbit.GetBindings("http://10.2.2.50")
	fmt.Println("Checking Bindings")
	for _, val := range bin {
		fmt.Printf("%+v    ::    %+v\n", val, err)
	}

	ex, err := rabbit.GetExchanges("http://10.2.2.50")

	for _, val := range ex {

	}*/
}

func addBold(s string) string {
	return fmt.Sprintf("\033[1m%s\033[0m", s)
}

func addGreen(s string) string {
	return fmt.Sprintf("\033[32m%s\033[0m", s)
}

func addRed(s string) string {
	return fmt.Sprintf("\033[31m%s\033[0m", s)
}
