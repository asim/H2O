package testmanager

import (
	"fmt"
	"github.com/HailoOSS/hshell/integrationtest/importstate"
	"github.com/HailoOSS/hshell/integrationtest/variables"
	"strconv"
	"sync"
	"time"
)

var Vars *variables.TestVariables
var wg sync.WaitGroup

func initVars() {
	Vars = variables.NewVariables()
}

var jj int = 0

func runTests(t func()) {
	fmt.Println("start")

	t()

	defer wg.Done()
	jj++
	fmt.Println(jj)
}

func StartTests(t func()) {

	err := importstate.GetVarsFromFile("/Users/jonathan/gopath/src/github.com/HailoOSS/hshell/integrationtest/test.conf")
	if err != nil {
		fmt.Println(err)
	}
	threads := variables.GlobalVar.GetVar("threads")
	rampup := variables.GlobalVar.GetVar("rampup")

	threadsint, err := strconv.Atoi(threads)
	if err != nil {
		threadsint = 1
	}

	rampupdur, err := time.ParseDuration(rampup)
	if err != nil {
		rampupdur = 0
	}

	for i := 0; i < threadsint; i++ {
		wg.Add(1)
		go runTests(t)
		time.Sleep(rampupdur)
	}

	wg.Wait()
}
