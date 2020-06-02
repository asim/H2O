package main

//creates a client and uses that to call a protobuf.
import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/HailoOSS/hshell/login"
	"github.com/HailoOSS/hshell/parseprotobuf"
	"github.com/HailoOSS/platform/client"
	"github.com/cihub/seelog"
	gouuid "github.com/nu7hatch/gouuid"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var tmpDir string
var setGoPath string
var latestTrace string

func main() {
	startCli := flag.Bool("cli", true, "starts the command line interface")
	tempGopath := flag.Bool("tempgopath", false, "use a temporary gopath or not")
	printTempDir := flag.Bool("printtempdir", false, "prints the location of the temporary directory")
	nuke := flag.Bool("nuke", false, "deletes the temp directory and redownloads everything")
	timeout := flag.String("timeout", "5000", "default timeout in hshell")
	retries := flag.Int("retries", 0, "default number of retries in hshell")
	live := flag.Bool("live", false, "If true this will connect via the bastion")
	staging := flag.Bool("staging", false, "If true this will connect via the bastion")
	bastion := flag.String("bastion", "", "The address of the bastion server to connect through")
	bastionUser := flag.String("bastionUser", "", "bastion username")
	env := flag.String("env", "", "The environment name you want to connect to (example: tst)")
	region := flag.String("region", "eu-west-1", "The region name you want to connect to (example: eu-west-1")
	flag.Parse()

	var err error

	// sets gopath to temp workspace (or not if requested)
	if *tempGopath || *printTempDir {
		tmpDir, err = CreateTempDir(*nuke) //creates a temporary workspace
		if err != nil {
			log.Println("error creating tempdir", err)
		}

		if *printTempDir {
			fmt.Println(tmpDir)
			return
		}
		origGoPath := os.Getenv("GOPATH")
		os.Setenv("GOPATH", tmpDir)
		defer os.Setenv("GOPATH", origGoPath)
	}
	setGoPath = os.Getenv("GOPATH")

	defaultTimeout, _ = time.ParseDuration(*timeout + "ms")
	defaultRetries = *retries

	//Starts the shell if requested
	//if the shell starts the program will never reach further than
	//this if statement
	if *startCli {
		cmdChan := make(chan PurlCommand)
		resultChan := make(chan string)
		go InteractiveShell(cmdChan, resultChan, *live, *staging, *bastion, *bastionUser, *env, *region)
		for cmd := range cmdChan {
			out := ""
			var err error
			service, endpnt := SeparateService(cmd.EndpointStr)
			out, err = SendJsonRequest(service, endpnt, cmd.JsonStr, cmd.Timeout, cmd.Retries)
			if err != nil {
				resultChan <- fmt.Sprintf("error running client: %+v \n%s\n", err, out)
				continue
			}
			resultChan <- fmt.Sprintf("%s\n", out)
		}
		return
	}
}

func GetJsonDefault(importStr string, protoName string) (string, error) {
	pbfile := fmt.Sprintf("%s/src/%s/%s.proto", setGoPath, importStr, protoName)

	file, err := os.Open(pbfile)
	if err != nil {
		log.Println("Could not read protos:", pbfile)
		return "", err
	}
	pb := parseprotobuf.ParseProtobufRaw(file, protoName, true, setGoPath) //PARSE PROTOBUF

	jsonStr, _ := parseprotobuf.PrintJsonExample(pb) //GET DEFAULT REQUEST

	return jsonStr, nil
}

//	Creates a temporary go directory in os.TempDir
func CreateTempDir(nuke bool) (string, error) {
	tmpDir := os.TempDir()
	if nuke {
		remerr := os.RemoveAll(tmpDir + "/go")
		if remerr != nil {
			seelog.Warnf("unable to delete temp dir: %v, error: %v", tmpDir+"/go", remerr)
		}
	}
	err := os.Mkdir(tmpDir+"/go", 0777)
	if err != nil {
		e, ok := err.(*os.PathError)
		if ok {
			if e.Err.Error() != "file exists" {
				return "", err
			}
		}
	}
	return tmpDir + "/go", nil
}

func SeparateService(service string) (string, string) {
	endpointStrArr := strings.Split(service, ".")
	endpointStr := endpointStrArr[len(endpointStrArr)-1]
	service = ""
	for i, val := range endpointStrArr {
		if i < len(endpointStrArr)-2 {
			service += val + "."
		} else if i < len(endpointStrArr)-1 {
			service += val
		}
	}
	return service, endpointStr
}

//sends a json request on to rabbit
func SendJsonRequest(service string, endpoint string, jsonString string, timeout time.Duration, retries int) (out string, err error) {
	req, opts, err := CreateRequest(service, endpoint, jsonString, timeout, retries)
	if err != nil {
		return
	}
	latestTrace = req.TraceID()
	rsp, dur, err := SendRequest(req, opts)
	if err != nil {
		out = fmt.Sprintf("Duration: %v\nTraceId: %s\n", dur.String(), latestTrace)
		return
	}

	var buf bytes.Buffer
	err = json.Indent(&buf, rsp.Body(), "", "  ")
	if err != nil {
		out = fmt.Sprintf("Duration: %v\nTraceId: %s\n", dur.String(), latestTrace)
		return
	}

	out = fmt.Sprintf("%s\n\nDuration: %v\nTraceId: %s\n", string(buf.Bytes()), dur.String(), latestTrace)
	return
}

//sends a json request on to rabbit
func SendRepeatedJsonRequest(service string, endpoint string, json string, timeout time.Duration, retries int, number int) (out string, err error) {
	req, opts, err := CreateRequest(service, endpoint, json, timeout, retries)
	if err != nil {
		return
	}
	var totalDuration time.Duration
	errNum := 0
	for i := 0; i < number; i++ {
		_, dur, err := SendRequest(req, opts)
		if err != nil {
			errNum++
		}
		totalDuration = totalDuration + dur

	}

	out = fmt.Sprintf("\nDuration: %v\nError Num: %v\n", totalDuration.String(), errNum)
	return
}

//sends a json request on to rabbit
func SendRepeatedAsyncJsonRequest(service string, endpoint string, json string, timeout time.Duration, retries int, number int) (out string, err error) {
	errNum := 0
	complete := make(chan bool)
	done := make(chan bool)

	go func() {
		for i := 0; i < number; i++ {
			if !<-done {
				errNum++
			}
		}
		complete <- true
	}()

	now := time.Now()
	for i := 0; i < number; i++ {
		go func() {
			req, opts, err := CreateRequest(service, endpoint, json, timeout, retries)
			if err != nil {
				//return
				done <- false
				return
			}
			_, _, err = SendRequest(req, opts)
			if err != nil {
				done <- false
				return
			}
			done <- true
		}()
	}
	<-complete
	totalDuration := time.Since(now)
	out = fmt.Sprintf("\nDuration: %v (%vs)\nThroughput: %v/s\nError Num: %v\n", totalDuration.Nanoseconds(), float64(totalDuration.Nanoseconds())*1.0e-9, float64(number)/(float64(totalDuration.Nanoseconds())*1.0e-9), errNum)
	return
}

func SendRequest(req *client.Request, opts client.Options) (*client.Response, time.Duration, error) {
	now := time.Now()
	rsp, err := client.CustomReq(req, opts)
	dur := time.Since(now)

	return rsp, dur, err
}

func CreateRequest(service string, endpoint string, json string, timeout time.Duration, retries int) (*client.Request, client.Options, error) {
	var req *client.Request
	req, err := client.NewJsonRequest(service, endpoint, []byte(json))
	if err != nil {
		return nil, nil, err
	}
	u4, _ := gouuid.NewV4()
	traceId := u4.String()
	req.SetTraceID(traceId)
	req.SetTraceShouldPersist(true)
	req.SetSessionID(login.Session)
	req.SetFrom(login.FromService)
	opts := client.Options{
		"timeout": timeout,
	}

	if retries != 0 {
		opts["retries"] = retries
	}

	return req, opts, nil
}

//runs go get on the import path
func GoGet(importStr string, update bool, args ...string) error {
	seelog.Info("Getting: " + importStr)

	var cmd *exec.Cmd

	updateArgs := []string{"get", "-u"}
	reuseArgs := []string{"get"}

	if update {
		for _, arg := range args {
			updateArgs = append(updateArgs, arg)
		}
		updateArgs = append(updateArgs, importStr)
		cmd = exec.Command("go", updateArgs...)
	} else {
		for _, arg := range args {
			reuseArgs = append(reuseArgs, arg)
		}
		reuseArgs = append(reuseArgs, importStr)
		cmd = exec.Command("go", reuseArgs...)
	}

	var outGet bytes.Buffer
	cmd.Stdout = &outGet
	var outErr bytes.Buffer
	cmd.Stderr = &outGet

	err := cmd.Start()
	if err != nil {
		seelog.Info(err)
	}

	err = cmd.Wait()
	if outGet.String() != "" {
		seelog.Info(outGet.String())
	}
	if outErr.String() != "" {
		seelog.Info(outErr.String())
	}
	if err != nil {
		seelog.Info("error: %v", err)
		return err
	}
	return nil
}

func GetDependancies(update bool) {
	//Get dependancies for client
	err := GoGet("github.com/HailoOSS/protobuf/proto", update)
	if err != nil {
		log.Println("problem getting code")
		return
	}

	err = GoGet("github.com/HailoOSS/platform/client", update)
	if err != nil {
		log.Println("problem getting code")
		return
	}

	err = GoGet("github.com/cihub/seelog", update)
	if err != nil {
		log.Println("problem getting code")
		return
	}
}
