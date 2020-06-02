package main

//This is used to power the consol like interface
import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/HailoOSS/gopass"
	"github.com/HailoOSS/hshell/binding"
	"github.com/HailoOSS/hshell/build"
	"github.com/HailoOSS/hshell/discovery"
	"github.com/HailoOSS/hshell/login"
	"github.com/HailoOSS/hshell/provision"
	vm "github.com/HailoOSS/hshell/versionmanager"
	"github.com/HailoOSS/platform/raven"
	seelog "github.com/cihub/seelog"
	"github.com/peterh/liner"
)

var initialCommands []Command = []Command{
	Command{Name: "list", Completer: serviceCompleter,
		HelpText: "list [service] Prints the services or endpoints depending on context"},
	Command{Name: "versions", Completer: serviceCompleter,
		HelpText: "versions [service] Prints the versions running of a given service"},
	Command{Name: "ls", Completer: serviceCompleter,
		HelpText: "ls [service] Prints the services or endpoints depending on context"},
	Command{Name: "use", Completer: serviceCompleter,
		HelpText: "use [service] sets the given service to be the active service. ie. the service you want to make calls from"},
	Command{Name: "cd", Completer: serviceCompleter,
		HelpText: "cd [service] sets the given service to be the active service. ie. the service you want to make calls from"},
	Command{Name: "stats", Completer: serviceCompleter,
		HelpText: "TODO DOES NOT CURRENTLY WORK"},
	Command{Name: "execute", Completer: endpointCompleter,
		HelpText: "execute [endpoint] [json] sends the given json to the given endpoint for the active service"},
	Command{Name: "env", Completer: envCompleter,
		HelpText: "env [environment] [region] connects to the designated environment and region"},
	Command{Name: "repeat", Completer: endpointCompleter,
		HelpText: "repeat [x] [endpoint] [json] sends the given json to the given endpoint for the active service x number of times"},
	Command{Name: "provision", Completer: serviceProvisionCompleter,
		HelpText: "provision [service] [machine class] [version] Provisions the given service"},
	Command{Name: "upgrade", Completer: serviceCompleter,
		HelpText: "upgrade [service] [version/branch] Provision the latest service. Allow for an optional version or branch name to be added. It will provision the latest version on that branch or the specific version you list"},
	Command{Name: "remove", Completer: removeServiceCompleter,
		HelpText: "remove [service] [machine class] [version] Removes the given service from the provisioned list"},
	Command{Name: "removeall", Completer: serviceCompleter,
		HelpText: "removeall [service] Removes all versions of the given service from the provisioned list"},
	Command{Name: "removeeverything", Completer: serviceCompleter,
		HelpText: "removeeverything Removes all services (USE AT YOUR OWN RISK)"},
	Command{Name: "weight", Completer: serviceWeightCompleter,
		HelpText: "weight [service] [version] [weight]adds weight to a service and version in the binding service"},
	Command{Name: "health", Completer: serviceCompleter,
		HelpText: "health [service] calls the healthcheck endpoint on a service. Must be logged in for call to work."},
	Command{Name: "session",
		HelpText: "session [session] manually set the token"},
	Command{Name: "service",
		HelpText: "service [service] allows hshell to pretend to be another service"},
	Command{Name: "shutdown", Completer: serviceCompleter,
		HelpText: "TODO DOES NOT CURRENTLY WORK"},
	Command{Name: "restart", Completer: removeServiceCompleter,
		HelpText: "restart [service] [class] [version] [azname] restarts a specific service in an az (az is optional)"},
	Command{Name: "restartaz", Completer: azCompleter,
		HelpText: "restartaz [azname] Restarts all the services in an AZ"},
	Command{Name: "import",
		HelpText: "import [file] imports a versions file and provisions those versions of the services"},
	Command{Name: "export",
		HelpText: "export [file] exports a versions file that can be imported elsewhere"},
	Command{Name: "timeout",
		HelpText: "timeout [ms] sets the default timeout for a call"},
	Command{Name: "h1login",
		HelpText: "h1login [username] gets a token to attach to every call. The dialog sometimes screws up. Just try again. Working on it..."},
	Command{Name: "login",
		HelpText: "login [username] gets a token to attach to every call. The dialog sometimes screws up. Just try again. Working on it... This is for H2."},
	Command{Name: "logout",
		HelpText: "logout deletes the current session (H2 only)."},
	Command{Name: "changepassword",
		HelpText: "changepassword [username] [password] [newpassword] changes your password"},
	Command{Name: "retries",
		HelpText: "retries [number] Sets the default number of retries"},
	Command{Name: "builds",
		HelpText: "TODO DOES NOTHING"},
	Command{Name: "quit"},
	Command{Name: "host",
		HelpText: "host [host] sets the rabbit host ie. 10.1.2.97"},
	Command{Name: "help",
		HelpText: "help help... who types help help?"},
}

var randomPort int = 55672
var randomStagingPort int = 55673
var defaultRabbitPort int = 5672

const (
	liveBastion    = "master-bastion01-live.i.HailoOSS.com"
	stagingBastion = "master-bastion01-staging.i.HailoOSS.com"
	envUrlTemplate = "%s.%s.i.%s.HailoOSS.net"
	sieUrlTemplate = "api-local.%s.e.hailoweb.com"
)

type Command struct {
	Name      string
	Completer func([]string, int) []Command
	HelpText  string
}

func protoCompleter(line []string, i int) []Command {

	protos := make([]string, 0)
	if currentPurl.EndpointStr != "" {
		protos = append(protos, discovery.GetProtobufForEndpoint(currentPurl.EndpointStr))
	}
	cmds := make([]Command, 0)
	for _, proto := range protos {
		if strings.HasPrefix(proto, line[i]) {
			cmds = append(cmds, Command{Name: proto})
		}
	}
	return cmds
}

func serviceEndpointCompleter(line []string, i int) []Command {
	comp := make([]Command, 0)
	services := discovery.GetServicesBuffered()
	for _, srv := range services {
		endpoints := discovery.GetEndpointsBuffered(srv)
		for _, endpoint := range endpoints {
			srvend := fmt.Sprintf("%s.%s", srv, endpoint)
			if strings.HasPrefix(srvend, line[i]) {
				comp = append(comp, Command{Name: srvend})
			}
		}

	}
	return comp
}

func azCompleter(line []string, i int) []Command {
	comp := make([]Command, 0)
	classArr := []string{
		"eu-west-1a",
		"eu-west-1b",
		"eu-west-1c",
	}

	for _, cls := range classArr {
		if strings.HasPrefix(cls, line[i]) {
			comp = append(comp, Command{Name: cls})
		}
	}

	return comp
}

func regionCompleter(line []string, i int) []Command {
	comp := make([]Command, 0)
	classArr := []string{
		"eu-west-1",
		"us-east-1",
		"ap-northeast-1",
	}

	for _, cls := range classArr {
		if strings.HasPrefix(cls, line[i]) {
			comp = append(comp, Command{Name: cls})
		}
	}

	return comp
}

func serviceProvisionCompleter(line []string, i int) []Command {

	services := build.GetAllBuilt()
	cmdSplit := strings.Split(line[i], ".")
	comp := dotServiceBuilder(services, cmdSplit)
	for i, _ := range comp {
		comp[i].Completer = classCompleter
	}
	return comp
}

func versionCompleter(line []string, i int) []Command {
	comp := make([]Command, 0)
	services := build.GetAllVersions(line[i-2])
	for _, srv := range services {
		if strings.HasPrefix(srv, line[i]) {
			comp = append(comp, Command{Name: srv})
		}
	}
	return comp
}

func envCompleter(line []string, i int) []Command {

	comp := make([]Command, 0)

	classArr := []string{
		"tst",
		"stg",
		"meta",
		"loadtest",
		"lve",
	}

	for _, cls := range classArr {
		if strings.HasPrefix(cls, line[i]) {
			comp = append(comp, Command{Name: cls, Completer: regionCompleter})
		}
	}

	return comp
}

func serviceVersionCompleter(line []string, i int) []Command {
	comp := make([]Command, 0)
	services := discovery.GetServiceVersionsBuffered(line[i-2])
	for _, srv := range services {
		verstr := strconv.FormatUint(srv.GetVersion(), 10)
		if strings.HasPrefix(verstr, line[i]) {
			comp = append(comp, Command{Name: verstr})
		}
	}
	return comp
}

func classCompleter(line []string, i int) []Command {
	comp := make([]Command, 0)
	classArr := discovery.GetMachineClassesBuffered("")

	for _, cls := range classArr {
		if strings.HasPrefix(cls, line[i]) {
			comp = append(comp, Command{Name: cls, Completer: versionCompleter})
		}
	}

	return comp
}

func serviceCompleter(line []string, i int) []Command {
	if !connected {
		connect("", 5672, liveMode, stagingMode, customMode, "")
		connected = true
	}
	services := discovery.GetServicesBuffered()
	cmdSplit := strings.Split(line[i], ".")

	comp := dotServiceBuilder(services, cmdSplit)

	return comp

}

func dotServiceBuilder(services, cmdSplit []string) []Command {

	comp := make([]Command, 0)

	addMap := make(map[string]bool)
	for _, srv := range services {
		discSplit := strings.Split(srv, ".")

		if len(cmdSplit) <= len(discSplit) {
			compName := ""
			add := true
			for ii, subCmd := range cmdSplit {
				if strings.HasPrefix(discSplit[ii], subCmd) {
					compName += discSplit[ii]
					if len(cmdSplit)-1 != ii || len(discSplit)-1 != ii {
						compName += "."
					}

				} else {
					add = false
				}
			}
			if add {
				addMap[compName] = true
			}
		}
	}
	for toadd, _ := range addMap {
		comp = append(comp, Command{Name: toadd})
	}

	return comp
}

func removeClassCompleter(line []string, i int) []Command {
	comp := classCompleter(line, i)
	for j, _ := range comp {
		comp[j].Completer = serviceVersionCompleter
	}
	return comp
}

func serviceWeightCompleter(line []string, i int) []Command {
	comp := serviceCompleter(line, i)
	for j, _ := range comp {
		comp[j].Completer = versionWeightCompleter
	}
	return comp
}

func versionWeightCompleter(line []string, i int) []Command {
	comp := make([]Command, 0)
	services := discovery.GetServiceVersionsBuffered(line[i-1])
	for _, srv := range services {
		verstr := strconv.FormatUint(srv.GetVersion(), 10)
		if strings.HasPrefix(verstr, line[i]) {
			comp = append(comp, Command{Name: verstr})
		}
	}
	return comp
}

func removeServiceCompleter(line []string, i int) []Command {
	comp := serviceCompleter(line, i)
	for j, _ := range comp {
		comp[j].Completer = removeClassCompleter
	}
	return comp
}

func endpointCompleter(line []string, i int) []Command {
	endpoints := []string{}
	if currentService != "" {
		endpoints = discovery.GetEndpointsBuffered(currentService)
	}

	cmds := make([]Command, 0)
	for _, endpoint := range endpoints {
		if strings.HasPrefix(endpoint, line[i]) {
			cmds = append(cmds, Command{Name: endpoint, Completer: jsonCompleter})
		}
	}
	return cmds
}

func jsonCompleter(line []string, i int) []Command {
	cmds := make([]Command, 0)
	json, _ := GetJsonDefault(currentPurl.ImportStr+"/proto/"+line[i-1], line[i-1])
	cmds = append(cmds, Command{Name: json})
	return cmds //cmds
}

func testComp(line []string, i int) []Command {
	return []Command{Command{Name: "test1"}}
}

func tabCompleter(line string) []string {
	resultchan := make(chan []string)

	go func(c chan []string) {
		lineArr := strings.Split(line, " ")
		previousCmd := Command{}
		strs := make([]string, 0)
		lineDepthStr := ""
		if len(lineArr) == 1 {

			for _, cmd := range initialCommands {
				if strings.HasPrefix(cmd.Name, line) {
					strs = append(strs, cmd.Name)
				}
			}

		}
		if len(lineArr) > 1 {

			for _, cmd := range initialCommands {
				if cmd.Name == lineArr[0] {
					previousCmd = cmd
				}
			}

			for i, ln := range lineArr {
				if i > 0 {
					if previousCmd.Completer != nil {
						cmds := previousCmd.Completer(lineArr, i)
						if len(lineArr)-1 == i {
							for _, cmd := range cmds {

								strs = append(strs, lineDepthStr+" "+cmd.Name)
							}
						} else {
							lineDepthStr += " " + ln
							for _, cmd := range cmds {
								if cmd.Name == ln {
									previousCmd = cmd
									break
								}
								previousCmd = Command{}
							}
						}
					}
				} else {
					lineDepthStr += ln
				}
			}

		}
		sort.Strings(strs)
		select {
		case c <- strs:
		default:
		}
		close(c)
	}(resultchan)

	select {
	case strs := <-resultchan:
		return strs
	case <-time.After(500 * time.Millisecond):
	}
	return make([]string, 0)
}

type PurlCommand struct {
	ImportStr   string
	JsonStr     string
	EndpointStr string
	AmqpHost    string
	Timeout     time.Duration
	Retries     int
}

var term *liner.State

var currentPurl PurlCommand
var currentService string = ""
var amqpHost string = ""
var defaultTimeout time.Duration = 5 * time.Second
var defaultRetries int
var connected bool = false
var liveMode bool = false
var stagingMode bool = false
var customMode string = ""
var cmdChannel = make(chan *exec.Cmd)
var historyFilename = path.Join(os.Getenv("HOME"), ".hshell_history")

func loadHistory() {
	if _, err := os.Stat(historyFilename); os.IsNotExist(err) {
		return
	}

	file, err := os.Open(historyFilename)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	_, err = term.ReadHistory(reader)
	if err != nil {
		fmt.Println(err)
	}
}

func saveHistory() {
	file, err := os.Create(historyFilename) // will open if it already exists
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = term.WriteHistory(writer)
	if err != nil {
		fmt.Println(err)
	}

	writer.Flush()
}

func closeTerm() {
	saveHistory()
	term.Close()
}

func exitHshell() {
	fmt.Println("exiting")
	closeTerm()
	for {
		select {
		case cmd := <-cmdChannel:
			cmd.Process.Kill()
		default:
			return
		}
	}
}

// @FIXME
// This function needs to be rewritten once live is migrated
func InteractiveShell(cmdChan chan PurlCommand, resultChan chan string, live bool, staging bool, bastion string, bastionUser string, env string, region string) {
	liveMode = live
	prefix := "> "
	colorpre := ""
	colorpost := ""
	switch {
	case live:
		prefix = "#LIVE> "
	case staging:
		prefix = "#STAGING> "
	}

	if len(bastion) > 0 {
		prefix = "#CUSTOM> "
		customMode = bastion
	}

	var rabbitmqHost string
	if len(env) > 0 && len(region) > 0 {
		if env == "lve" || env == "live" {
			colorpre = "\033[1;31m"
			colorpost = "\033[0m"
			prefix = fmt.Sprintf("#%v(%v)> ", strings.ToUpper(env), region)
		} else {
			colorpre = ""
			colorpost = ""
			prefix = fmt.Sprintf("#%v(%v)> ", strings.ToUpper(env), region)
		}
		bastion, rabbitmqHost = buildEnvUrls(env, region)
	}

	if err := checkAccess(live, staging, bastion); err != nil {
		fmt.Printf("Unable to connect to bastion\n")
		close(cmdChan)
		return
	}

	stagingMode = staging
	prevLogger := seelog.Current
	seelog.ReplaceLogger(seelog.Disabled)
	log.SetOutput(ioutil.Discard)
	term = liner.NewLiner()
	defer func() {
		exitHshell()
		close(cmdChan)
	}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func(c chan os.Signal) {
		for _ = range c {
			fmt.Println("ctrl-c caught, exiting \033[0m")
			closeTerm()
			close(cmdChan)
			close(c)
		}
	}(sigChan)

	fmt.Println("h2o Interactive Shell")

	loadHistory()

	term.SetCompleter(tabCompleter)
	promptPrefix := ""
	for {
		// @FIXME
		// Needs to be refactored once everything is migrated to the new env
		if !connected && len(rabbitmqHost) > 0 {

			connect(rabbitmqHost, 5672, live, staging, bastion, bastionUser)
			connected = true
		}
		fmt.Printf(colorpre)
		l, e := term.Prompt(promptPrefix + prefix)
		fmt.Printf(colorpost)
		if e != nil {
			fmt.Printf("Error, quitting: %v", e)
			break
		}

		s := string(l)
		s = strings.TrimSpace(s)
		parts := strings.Fields(s)
		validCommand := true

		if len(parts) == 0 {
			continue
		}

		if (parts[0] != "host" && parts[0] != "logs" && parts[0] != "help" && parts[0] != "env") && !connected {
			connect("", 5672, live, staging, bastion, bastionUser)
			connected = true
		}

		switch parts[0] {
		case "quit", "exit":
			return
		case "execute":
			if len(parts) >= 3 {
				jsonStr := collapseToString(parts[2:])

				sendingPurl := PurlCommand{EndpointStr: currentService + "." + parts[1],
					ImportStr: currentPurl.ImportStr + "/proto/" + parts[1],
					JsonStr:   jsonStr,
					AmqpHost:  amqpHost,
					Timeout:   defaultTimeout,
					Retries:   defaultRetries}

				cmdChan <- sendingPurl
				fmt.Println(<-resultChan)
			} else {
				invalidSyntaxError()
			}
		case "health":
			healthServ := currentService
			if len(parts) == 2 {
				healthServ = parts[1]
			}

			sendingPurl := PurlCommand{EndpointStr: healthServ + ".health",
				JsonStr:  "{}",
				AmqpHost: amqpHost,
				Timeout:  defaultTimeout,
				Retries:  defaultRetries}
			cmdChan <- sendingPurl
			fmt.Println(<-resultChan)
		case "repeat":
			if len(parts) >= 4 {
				num, err := strconv.Atoi(parts[1])
				if err != nil {
					invalidSyntaxError()
					continue
				}

				jsonStr := collapseToString(parts[3:])

				out, err := SendRepeatedAsyncJsonRequest(currentService, parts[2], jsonStr, defaultTimeout, defaultRetries, num)
				fmt.Println(out, err)
			} else {
				invalidSyntaxError()
			}
		case "restart":
			if len(parts) >= 4 {

				if len(login.Session) == 0 {
					fmt.Println("WARNING: Your session is empty. This will likely fail.")
				}

				azName := ""
				if len(parts) == 5 {
					azName = parts[4]
				}

				vernum, err := strconv.ParseUint(parts[3], 10, 64)
				if err != nil {
					fmt.Println("Not a valid version number:", parts[3], err)
					continue
				}

				err = provision.PubRestart(parts[1], parts[2], vernum, azName)
				if err != nil {
					fmt.Printf("Could not restart service %v\n", err)
				} else {
					fmt.Printf("Restarting service %s, %s on class %s in az %s\nThere is a 60s jitter on the restart\n",
						parts[1], parts[3], parts[2], azName)
				}
			} else {
				invalidSyntaxError()
			}
		case "restartaz":
			azName := ""
			if len(parts) == 2 {
				azName = parts[1]
			}

			err := provision.PubRestartAz(azName)
			if err != nil {
				fmt.Printf("Could not restart services %v\n", err)
			} else {
				fmt.Printf("Restarted services in %s\n", azName)
			}

		case "cd", "use":
			if len(parts) == 2 {
				if parts[1] == ".." {
					currentService = ""
					promptPrefix = ""
				} else {
					git := discovery.GetServiceGit(parts[1])
					currentPurl = PurlCommand{ImportStr: git}
					if len(git) == 0 {
						fmt.Println("Error: service does not specify source url. Autocomplete will fail")
					} else {
						fmt.Println("Downloading ", git)
					}

					GoGet(git, false, "-d")
					currentService = parts[1]
					promptPrefix = currentService
				}
			} else if len(parts) == 1 {
				currentService = ""
				promptPrefix = ""
			} else {
				validCommand = false
				invalidSyntaxError()
			}
		case "status", "stats":
			if len(parts) == 1 && currentService != "" {
				printStats(currentService)
			} else if len(parts) == 2 {
				printStats(parts[1])
			} else {
				invalidSyntaxError()
			}

		case "ls", "list":
			if len(parts) == 1 {
				if currentService != "" {
					printEndpoints(currentService)
				} else {
					services := discovery.GetServicesBuffered()
					fmt.Println("Services:")
					for _, srv := range services {
						fmt.Printf("   %s\n", srv)
					}
					fmt.Println("")
				}
			} else {
				printEndpoints(parts[1])
			}
		case "weight":
			if len(parts) == 4 {
				weight, _ := strconv.Atoi(parts[3])
				binding.CreateRule(parts[1], parts[2], weight)
			} else if len(parts) == 3 {
				if parts[2] == "remove" {
					rules, _ := binding.ListRules(parts[1])
					for _, rule := range rules {
						binding.DeleteRule(rule.GetService(), rule.GetVersion(), int(rule.GetWeight()))
					}
				}
			} else {
				invalidSyntaxError()
			}
		case "help":
			printHelp(parts)
		case "host":
			if len(parts) == 2 {
				amqpHost = parts[1]
				discovery.AmqpHost = amqpHost
				hostArr := strings.Split(amqpHost, ":")
				port := defaultRabbitPort
				if len(hostArr) == 2 {
					amqpHost = hostArr[0]
					port, _ = strconv.Atoi(hostArr[1])
				}
				connect(amqpHost, port, live, staging, bastion, bastionUser)
				connected = true
			} else {
				fmt.Println("Host:", amqpHost)
			}
		case "env":
			if len(parts) >= 2 {
				if len(parts) < 3 {
					if region == "" {
						region = "eu-west-1"
					}
					fmt.Printf("Region is not specified, defaulting to %v\n", region)
				} else {
					region = parts[2]
				}
				env = parts[1]

				select {
				case cmd, ok := <-cmdChannel:
					if ok {
						cmd.Process.Kill()
					}
				default:
					// Do nothing
				}

				bastion, amqpHost = buildEnvUrls(env, region)
				discovery.AmqpHost = amqpHost
				if env == "lve" || env == "live" {
					colorpre = "\033[1;31m"
					colorpost = "\033[0m"
					prefix = fmt.Sprintf("#%v(%v)> ", strings.ToUpper(env), region)
				} else {
					colorpre = ""
					colorpost = ""
					prefix = fmt.Sprintf("#%v(%v)> ", strings.ToUpper(env), region)
				}
				port := defaultRabbitPort
				connect(amqpHost, port, live, staging, bastion, bastionUser)
				connected = true

			} else {
				invalidSyntaxError()
			}
		case "lsweight":
			verServ := ""
			if len(parts) == 2 {
				verServ = parts[1]
			} else if len(parts) == 1 && len(currentService) != 0 {
				verServ = currentService
			}

			services := discovery.GetServiceVersionsBuffered(verServ)

			fmt.Print("Running versions ")

			if len(verServ) > 0 {
				fmt.Println("for", verServ)
			} else {
				fmt.Println()
			}

			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 0, 1, 1, ' ', 0)
			for _, service := range services {
				ver := strconv.FormatUint(service.GetVersion(), 10)
				branch, hash := build.GetBranchBuffered(service.GetName(), ver)
				weight := binding.ListRulesBuffered(service.GetName())
				weightStr := ""
				if len(weight) > 0 {
					for _, we := range weight {
						if we.GetVersion() == ver {
							weightStr = "Weight: " + strconv.Itoa(int(we.GetWeight()))
						}
					}
				}
				if len(verServ) == 0 {
					fmt.Fprint(w, " \t"+service.GetName())
				}
				fmt.Fprintln(w, " \t"+ver+"\t"+branch+"\t"+hash+"\t"+weightStr)
			}
			fmt.Fprintln(w)
			w.Flush()
		case "removeeverything":
			if !live && env != "lve" {
				servs, err := provision.GetServiceVersionMachineClass()
				if err != nil {
					fmt.Println("problem finding services")
					continue
				}
				for _, serv := range servs {
					if !strings.Contains(serv.Service, "provisioning") &&
						!strings.Contains(serv.Service, "discovery") &&
						!strings.Contains(serv.Service, "binding") &&
						!strings.Contains(serv.Service, "config") &&
						!strings.Contains(serv.Service, "login") {

						err := provision.DeleteService(serv.Service, serv.MachineClass, serv.Version)
						if err != nil {
							fmt.Println("Error removing service: ", err)
						} else {
							fmt.Printf("Removed %s:%s on machine class %s\n", serv.Service, serv.MachineClass, serv.Version)
						}
					}
				}
			} else {
				fmt.Println("Not allowed in live because of DANGER ZONE!")
			}
		case "versions":
			verServ := ""
			if len(parts) == 2 {
				verServ = parts[1]
			} else if len(parts) == 1 && len(currentService) != 0 {
				verServ = currentService
			}

			services := discovery.GetServiceVersionsBuffered(verServ)
			machineClasses, _ := provision.GetServiceVersionMachineClass()

			fmt.Print("Running versions ")

			if len(verServ) > 0 {
				fmt.Println("for", verServ)
			} else {
				fmt.Println()
			}

			w := new(tabwriter.Writer)
			w.Init(os.Stdout, 0, 1, 1, ' ', 0)

			if len(verServ) == 0 {
				fmt.Fprint(w, "\tName")
			}
			fmt.Fprintln(w, " \tVersion\tHash\tBranch\tMachine Classes")

			for _, service := range services {
				ver := strconv.FormatUint(service.GetVersion(), 10)
				branch, hash := build.GetBranchBuffered(service.GetName(), ver)

				mc := make([]string, 0)
				for _, machineClass := range machineClasses {
					if machineClass.Service == service.GetName() && machineClass.Version == int(service.GetVersion()) {
						mc = append(mc, machineClass.MachineClass)
					}
				}

				if len(verServ) == 0 {
					fmt.Fprint(w, " \t"+service.GetName())
				}
				fmt.Fprintln(w, " \t"+ver+"\t"+hash+"\t"+branch+"\t"+strings.Join(mc, ","))
			}
			fmt.Fprintln(w)
			w.Flush()
		case "provision":
			if len(parts) == 4 {
				version, _ := strconv.Atoi(parts[3])
				err := provision.CreateService(parts[1], parts[2], version)
				if err != nil {
					fmt.Println("Problem provisioning service: ", err)
				}
			} else {
				fmt.Println("Invalid Syntax")
			}
		case "upgrade":
			// THIS NEEDS A REFACTOR
			if !live && env != "lve" {
				if len(parts) >= 1 {
					service := currentService
					version := "€€@#$~|?ºª•¶§∞∞¢#€¡" // Assume no branch is ever named this... cruft

					if len(parts) >= 2 {
						service = parts[1]
					}
					if len(parts) >= 3 {
						// This could be a version number or a branch name
						version = parts[2]
					}

					if len(service) > 0 {
						cVersions := discovery.GetServiceVersions(service)

						versionint, err := strconv.Atoi(version)
						if err != nil || versionint == 0 {
							// The provided version is either 0 or not an int
							// it might be a branch name, this won't work if a branch name is an int...
							versionint = build.GetLatestVersionBranch(service, version)
							// reset the error
							err = nil
						}
						if versionint == 0 {
							version = build.GetLatestVersion(service)
							versionint, err = strconv.Atoi(version)
						}
						if err != nil || versionint == 0 {
							fmt.Printf("Invalid version: %v", version)
						} else {
							classArr := provision.GetServiceMachineClasses(service)
							for _, class := range classArr {
								err := provision.CreateService(service, class, versionint)
								if err != nil {
									fmt.Println("Problem provisioning service: ", err)
									continue
								}

								for _, val := range cVersions {
									if val.GetVersion() != uint64(versionint) {
										err = provision.DeleteService(service, class, int(val.GetVersion()))
										if err != nil {
											fmt.Printf("Could not remove old version %v on machine class %s\n", val.GetVersion(), class)
										}
									}
								}
								fmt.Printf("Upgraded %s to version %v on machine class %s\n", service, versionint, class)
							}
						}
					}
				} else {
					fmt.Println("Invalid Syntax")
				}
			} else {
				fmt.Println("Not allowed in production!")
			}
		case "upgradeall":
			if len(parts) == 1 {

				cVersions := discovery.GetServiceVersions("")

				for _, val := range cVersions {
					classArr := provision.GetServiceMachineClasses(val.GetName())
					version := build.GetLatestVersion(val.GetName())
					versionint, _ := strconv.Atoi(version)
					for _, class := range classArr {
						if versionint != int(val.GetVersion()) {
							err := provision.CreateService(val.GetName(), class, versionint)
							if err != nil {
								fmt.Println("Problem provisioning service: ", err)
							}
							for _, valToDel := range cVersions {
								if valToDel.GetName() == val.GetName() && int(valToDel.GetVersion()) != versionint {
									provision.DeleteService(val.GetName(), class, int(valToDel.GetVersion()))
								}
							}
							fmt.Printf("Upgraded %s to version %v\n", val.GetName(), val.GetVersion())
						}
					}
				}

			} else {
				fmt.Println("Invalid Syntax")
			}
		case "h1login":

			if len(parts) == 3 {
				token, err := login.Login(parts[1], parts[2])
				if err != nil {
					fmt.Println("Login Problem:", err)
				} else {
					fmt.Println("Login Success:", token)
				}
			} else if len(parts) == 2 {
				saveHistory()
				term.Close()
				pass, err := gopass.GetPass("Input Password: ")
				term = liner.NewLiner()
				loadHistory()
				term.SetCompleter(tabCompleter)
				if err != nil {
					fmt.Println("Unable to read password from prompt: ", err)
				}

				token, err := login.Login(parts[1], pass)

				if err != nil {
					fmt.Println("Login Problem:", err)
				} else {
					fmt.Println("Login Success:", token)
				}
			} else {
				invalidSyntaxError()
			}
		case "login":

			if len(parts) == 3 {
				token, err := login.LoginH2(parts[1], parts[2])
				if err != nil {
					fmt.Println("Login Problem:", err)
				} else {
					fmt.Println("Login Success:", token)
				}
			} else if len(parts) == 2 {
				saveHistory()
				term.Close()
				pass, err := gopass.GetPass("Input Password: ")
				term = liner.NewLiner()
				loadHistory()
				term.SetCompleter(tabCompleter)
				if err != nil {
					fmt.Println("Unable to read password from prompt: ", err)
				}

				token, err := login.LoginH2(parts[1], pass)

				if err != nil {
					fmt.Println("Login Problem:", err)
				} else {
					fmt.Println("Login Success:", token)
				}
			} else {
				invalidSyntaxError()
			}
		case "logout":
			if login.Session != "" {
				if err := login.Logout(); err != nil {
					fmt.Println("Logout error: ", err)
				} else {
					fmt.Println("Logged out")
				}
			} else {
				fmt.Println("Not logged in")
			}
		case "session":
			if len(parts) == 2 {
				login.Session = parts[1]
				fmt.Printf("Session set to %s\n", parts[1])
			} else {
				invalidSyntaxError()
			}
		case "service":
			if len(parts) == 2 {
				login.FromService = parts[1]
				fmt.Printf("Service set to %s\n", parts[1])
			} else {
				invalidSyntaxError()
			}
		case "changepassword":
			if len(parts) == 4 {
				token, err := login.NewPasswordH2(parts[1], parts[2], parts[3])
				if err != nil {
					fmt.Println("Change password error:", err)
				} else {
					fmt.Println("Change password successful:", token)
				}
			} else {
				invalidSyntaxError()
			}
		case "h1changepassword":
			if len(parts) == 4 {
				token, err := login.NewPassword(parts[1], parts[2], parts[3])
				if err != nil {
					fmt.Println("Change password error:", err)
				} else {
					fmt.Println("Change password successful:", token)
				}
			} else {
				invalidSyntaxError()
			}
		case "shutdown":
			//TODO
		case "remove":
			if len(parts) == 4 {
				version, _ := strconv.Atoi(parts[3])
				err := provision.DeleteService(parts[1], parts[2], version)
				if err != nil {
					fmt.Println("Problem provisioning service: ", err)
				}
			} else {
				invalidSyntaxError()
			}
		case "removeall":
			if len(parts) == 2 {
				classArr := provision.GetServiceMachineClasses(parts[1])
				services := discovery.GetServiceVersionsBuffered(parts[1])
				for _, service := range services {
					for _, class := range classArr {
						err := provision.DeleteService(parts[1], class, int(service.GetVersion()))
						if err != nil {
							fmt.Println("Problem provisioning service: ", err)
						}
					}
				}
			} else {
				invalidSyntaxError()
			}
		case "export":
			if len(parts) == 2 {
				filename := parts[1]
				err := vm.ExportVersions(filename)
				if err != nil {
					fmt.Println(err)
				} else {
					fmt.Println("Exported current state to file: ", filename)
				}

			} else {
				invalidSyntaxError()
			}
		case "import":
			if !live && env != "lve" {
				if len(parts) == 2 {
					filename := parts[1]
					err := vm.ImportVersions(filename)
					if err != nil {
						fmt.Println(err)
					}
				} else {
					invalidSyntaxError()
				}
			} else {
				fmt.Println("Not allowed in live because of DANGER ZONE!")
			}
		case "timeout":
			if len(parts) == 2 {
				dur, err := time.ParseDuration(parts[1] + "ms")
				if err == nil {
					defaultTimeout = dur
				}
			} else if len(parts) == 1 {
				fmt.Printf("Timeout: %v\n", float64(defaultTimeout.Nanoseconds())*1e-6)
			}
		case "retries":
			if len(parts) == 2 {
				ret, err := strconv.Atoi(parts[1])
				if err == nil {
					defaultRetries = ret
				}
			} else if len(parts) == 1 {
				fmt.Printf("Retries: %v\n", defaultRetries)
			}
		case "builds":
			//TODO
		case "logs":
			if len(parts) == 2 {
				if parts[1] == "enable" {
					seelog.ReplaceLogger(prevLogger)
				}
				if parts[1] == "disable" {
					seelog.ReplaceLogger(seelog.Disabled)
				}
			}
		case "":
			validCommand = false
		default:
			fmt.Println("Invalid Command")
			validCommand = false
		}
		if validCommand {
			term.AppendHistory(s)
		}

	}
}

func invalidSyntaxError() {
	fmt.Println("Invalid Syntax")
}

func setEnvironment(host string, localport int, remoteport int, bastion string, bastionUser string, successMessage string) {
	if bastionUser != "" {
		bastion = fmt.Sprintf("%s@%s", bastionUser, bastion)
	}
	cmd := exec.Command("ssh", "-o StrictHostKeyChecking=no", "-N", "-T", "-L", fmt.Sprintf("%v:%v:%v", localport, host, remoteport), bastion)

	err := cmd.Start()
	if err != nil {
		log.Println(err)
	}
	cmdChannel <- cmd
	err = cmd.Wait()
	fmt.Println(successMessage)
}

func connect(host string, port int, live bool, staging bool, bastion string, bastionUser string) {
	if live || staging || len(bastion) > 0 {
		fmt.Println("Connecting to bastion (this might take some time)")
		if live {
			go setEnvironment(host, randomPort, port, liveBastion, bastionUser, "Connected To Live Bastion")
			port = randomPort
		}
		if staging {
			go setEnvironment(host, randomStagingPort, port, stagingBastion, bastionUser, "Connected to Staging Bastion")
			port = randomStagingPort
		}
		if len(bastion) > 0 {
			rand.Seed(time.Now().Unix())
			randInt := rand.Intn(10000)
			randInt += 10000
			go setEnvironment(host, randInt, port, bastion, bastionUser, fmt.Sprintf("Connected to custom Bastion: %s:%v", bastion, randInt))
			port = randInt
		}
		host = "localhost"
	}
	if len(host) != 0 {
		raven.AmqpUri = fmt.Sprintf("amqp://hailo:hailo@%v:%v", host, port)
	}
	fmt.Println("Connecting to", raven.AmqpUri)
	<-raven.Connect()
}

func collapseToString(arr []string) string {
	spcStr := ""
	jsonStr := ""
	for _, str := range arr {
		jsonStr = jsonStr + spcStr + str

		cnt := strings.Count(str, `"`)
		subcnt := strings.Count(str, `\"`)
		cnt = cnt - subcnt
		if cnt%2 == 1 {
			if spcStr == "" {
				spcStr = " "
			} else {
				spcStr = ""
			}
		}
	}
	return jsonStr
}

func printStats(service string) {

	toPrint := "\n" + service + ` Stats:
  Num. Instances:               3
  Last Instance Joined:         25/06/2013 11:13:21
  Num Recent Errors (10 min):   6
  Auto Scaling Group Class:     SomeClass
  Owner:                        Jono
  Source:                       github.com/HailoOSS/Repo
  Highest Version:              123456789
  Lowest Version:               000000001

`
	fmt.Printf(toPrint)

}

func printEndpoints(service string) {
	endpoints := discovery.GetEndpointsBuffered(service)
	fmt.Printf("Endpoints for %s:\n", service)
	for _, srv := range endpoints {
		fmt.Printf("   %s\n", srv)
	}
	fmt.Println("")
}

func printHelp(parts []string) {
	if len(parts) == 1 {
		fmt.Println("Commands:")
		cmdNames := make([]string, len(initialCommands))
		for i, cmd := range initialCommands {
			cmdNames[i] = cmd.Name
		}
		sort.Strings(cmdNames)
		for _, cmd := range cmdNames {
			fmt.Printf("\t%s\n", cmd)
		}
	} else if len(parts) == 2 {
		for _, cmd := range initialCommands {
			if cmd.Name == parts[1] {
				fmt.Printf("Help for %s\n", cmd.Name)
				fmt.Printf("\t%s\n", cmd.HelpText)
			}
		}
	}
}

func checkAccess(live, staging bool, bastion string) error {
	if live {
		bastion = liveBastion
	} else if staging {
		bastion = stagingBastion
	} else if len(bastion) == 0 {
		return nil
	}
	con, err := net.DialTimeout("tcp", fmt.Sprintf("%s:22", bastion), time.Second*15)
	if err != nil {
		return err
	}
	con.Close()
	return nil
}

// dnsLookUp returns the address of a particular host
func dnsLookUp(host string) ([]string, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	var hosts []string

	for _, ip := range ips {
		hosts = append(hosts, ip.String())
	}

	sort.Strings(hosts)

	return hosts, nil
}

// buildEnvUrls returns the url mappings required for establishing an env connection
func buildEnvUrls(env, region string) (string, string) {
	if env == "" || region == "" {
		return "", ""
	}
	bastionUrl := fmt.Sprintf(envUrlTemplate, "bastion", region, env)
	rabbitUrl := fmt.Sprintf(envUrlTemplate, "rabbitmq", region, env)
	// Check if the bastion server exists
	// We assume that if it doesn't, we deal with single instance environment
	ips, _ := dnsLookUp(bastionUrl)
	if len(ips) == 0 {
		bastionUrl = ""
		rabbitUrl = fmt.Sprintf(sieUrlTemplate, env)
	}

	return bastionUrl, rabbitUrl
}
