package main

//creates a client and uses that to call a protobuf.
import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/HailoOSS/hshell/send"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const (
	GitProtoHome   = "github.com/HailoOSS/protobufs/services"
	IMPORT         = "%IMPORT%"
	REQUEST        = "%REQUEST%"
	RESPONSE       = "%RESPONSE%"
	ENDPOINT       = "%ENDPOINT%"
	OUTPUT         = "%OUTPUT%"
	REQUESTUPDATE  = "%REQUESTUPDATE%"
	SERVICE        = "%SERVICE%"
	CLIENTTEMPLATE = `
package main

import (
	"github.com/HailoOSS/protobuf/proto"	
	"github.com/HailoOSS/platform/client"
	"time"
	"os"
	"strings"
	log "github.com/cihub/seelog"
	"flag"
	"fmt"
	%IMPORT%
)
var _ = proto.String("NOOP")

func init() {
	logger, err := log.LoggerFromConfigAsFile("seelog.xml")

	if err != nil {
		log.Debugf("Failed to load logger config from seelog.xml %v", err)
	} else {
		log.ReplaceLogger(logger)
		log.Info("Custom logging enabled")
	}
}

func main() {
	defer log.Flush()

	reqNumber := flag.Int("number", 1, "Number of iterations")
	outFile := flag.String("output", "", "Location of the output log")
	flag.Parse()

	req := %REQUEST%
	endpoint := "%ENDPOINT%"
	service := "%SERVICE%"

	fo, err := os.Create(*outFile)
    if err != nil { 
    	log.Error("output file creation failed:", err) 
    	return 
    }
    // close fo on exit and check for its returned error
    defer func() {
        if err := fo.Close(); err != nil {
            log.Error("Cannot close output file:", err) 
            return
        }
    }()
      
    writeChan := make(chan MessageData)   
    done := make(chan int)

    go WriteOutput(fo, writeChan, done)

	for i :=0; i < *reqNumber; i++ {
		%REQUESTUPDATE%
		request, err := client.NewRequest(
			service,
			endpoint,
			req,
		)
		if err != nil {
			log.Error(err)
			return
		}

		rsp := &%RESPONSE%

		ts := time.Now()
		errReq := client.Req(request, rsp)
		duration := time.Since(ts)
		if errReq != nil {			
			writeChan <- MessageData{
				ep: service +"."+endpoint,
				rt: duration,
				status: "false",
				rs: 0,
				err: errReq,
				h: "",
				r: "",
			}
			continue
		}
		
		%OUTPUT%		
		writeChan <- MessageData{
			ep: service +"."+endpoint,
			rt: duration,
			status: "true",
			rs: 0,
			err: err,			
			r: rsp,
		}
	}
	close(writeChan)
	<-done
}

type MessageData struct {
	ep     string
	rt     time.Duration
	status string
	rs     int
	err    ErrorResponse
	h      Header
	r      Response
}

type Header interface{}
type Response interface{}
type ErrorResponse interface{}

func WriteOutput(fo *os.File, writeChan chan MessageData, done chan int) {
	for md := range writeChan {	
		toWrite := ""
		results := fmt.Sprintf("%+v", md.r)
		header := fmt.Sprintf("%+v", md.h)
		results = fmt.Sprintf("\"%s\"", strings.Replace(results, "\"", "\"\"", -1))
		header = fmt.Sprintf("\"%s\"", strings.Replace(header, "\"", "\"\"", -1))
		ts := time.Now()
		toWrite += fmt.Sprintf("%v,", ts.UnixNano())
		toWrite += md.ep + ","
		toWrite += fmt.Sprintf("%v,", md.rt.Nanoseconds())
		toWrite += md.status + ","
		toWrite += string(md.rs) + ","
		if md.err != nil {
			toWrite += fmt.Sprintf("%+v,", md.err)
		} else {
			toWrite += ","
		}
		toWrite += fmt.Sprintf("%+v,", results)
		toWrite += fmt.Sprintf("%+v", header)
		fo.WriteString(toWrite + "\n")
	}
	done <- 1
}
`
)

func main() {

	outputFile := flag.String("output", "test-client.go", "Output filename")
	importStr := flag.String("protobuf", "", "The import protobuf path, ie. go-banning-service/audit")
	requestStr := flag.String("go", "", `The request object in native Go ie. "Name: proto.String(\"Moddie\"),"`)
	endpointStr := flag.String("endpoint", "", "The endpoint to hit ie. com.HailoOSS.banning.retrieve")
	jsonStr := flag.String("json", "", "the request object as json")
	hint := flag.Bool("hint", false, "Give you a peek at the protobuf.  Won't actually run the request")
	defaultReq := flag.Bool("default", false, "if true, send the default request")
	update := flag.Bool("update", false, "update Go files. That is the protobuf, the protobuf library and the client library")
	flag.Parse()

	tmpDir, err := send.CreateTempDir()
	if err != nil {
		log.Println("error creating tempdir", err)
	}

	origGoPath := os.Getenv("GOPATH")
	os.Setenv("GOPATH", tmpDir)
	defer os.Setenv("GOPATH", origGoPath)

	protoArr := strings.Split(*importStr, "/")

	err = send.GoGet(*importStr, *update)
	if err != nil {
		log.Println("problem getting code")
		return
	}

	pbfile := fmt.Sprintf("%s/src/%s/%s.proto", tmpDir, *importStr, protoArr[len(protoArr)-1])

	data, err := ioutil.ReadFile(pbfile)
	if err != nil {
		log.Println("Could not read protos:", pbfile)
	}
	pbreader := bytes.NewReader(data)
	pbioreader := bufio.NewReader(pbreader)

	pb := send.ParseProtobufRaw(pbioreader, protoArr[len(protoArr)-1], true, tmpDir)

	if *hint {
		send.PrintHint(pb)
		send.PrintJsonExample(pb)
		return
	}

	if *defaultReq {
		*jsonStr, _ = send.PrintJsonExample(pb)
	}

	var req string
	if *jsonStr != "" {
		req = send.ConstructJsonRequest(pb, *jsonStr)
		for _, pbVal := range pb {
			if pbVal.Root {
				*importStr = pbVal.Name + " \"" + *importStr + "\"\n\"encoding/json\""
			}
		}
	} else {
		req = send.ConstructRequest(pb, *requestStr)
		for _, pbVal := range pb {
			if pbVal.Root {
				*importStr = pbVal.Name + " \"" + *importStr + "\""
			}
		}
	}
	rsp := send.ConstructResponse(pb)

	outputStr := ""

	service, endpoint := send.SeparateService(*endpointStr)

	crArr := []send.ClientReplace{
		send.ClientReplace{Find: IMPORT, Replace: *importStr},
		send.ClientReplace{Find: REQUEST, Replace: req},
		send.ClientReplace{Find: RESPONSE, Replace: rsp},
		send.ClientReplace{Find: ENDPOINT, Replace: endpoint},
		send.ClientReplace{Find: OUTPUT, Replace: outputStr},
		send.ClientReplace{Find: SERVICE, Replace: service},
		send.ClientReplace{Find: REQUESTUPDATE, Replace: ""},
	}

	rdrstr := strings.NewReader(CLIENTTEMPLATE)
	rdr := bufio.NewReader(rdrstr)

	_, err = send.WriteClient(rdr, crArr, *outputFile)
	if err != nil {
		log.Println("Error writing client:", err)
	}

}
