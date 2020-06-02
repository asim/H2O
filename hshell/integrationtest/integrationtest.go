package main

import (
	"fmt"
	seelog "github.com/cihub/seelog"
	li "github.com/HailoOSS/hshell/integrationtest/loginintegration"
	"github.com/HailoOSS/hshell/integrationtest/result"
	"github.com/HailoOSS/hshell/integrationtest/testmanager"
	"github.com/HailoOSS/hshell/integrationtest/variables"
	"io/ioutil"
	"log"
	"runtime"
	"strconv"
)

func init() {
	seelog.ReplaceLogger(seelog.Disabled)
	log.SetOutput(ioutil.Discard)
}

func main() {
	runtime.GOMAXPROCS(8)
	//variables.GlobalVar.SetVar("threads", "4")
	//variables.GlobalVar.SetVar("rampup", "5s")
	testmanager.StartTests(Integration)

	res := result.GetResults()

	for name, val := range res {
		fmt.Printf("%+v: %+v\n", name, val)
	}
}

/*func LogoutDrivers() {
	for i := 0; i < 500; i++ {
		tv := variables.NewVariables()
		istr := strconv.Itoa(i + 21100)
		postData := map[string]string{
			"keywords": "ltdrive-" + istr + "@example.com",
		}
		rsp, _ := request.DoHttpRequest("search",
			driverHost,
			"/v1/driver/search/",
			"GET",
			postData,
			validators.RegexValidator(`{"status":true,"payload":{`),
		)
		tv.SetVarRegex(`"id":"([^"]+)"`, "driver_id", string(rsp))
		postData = map[string]string{
			"driver": tv.GetVar("driver_id"),
		}
		_, err := request.DoHttpRequest("Logout",
			driverHost,
			"/v1/driver/logout/",
			"POST",
			postData,
			validators.RegexValidator(`{"status":true,"payload":{`),
		)
		fmt.Println(err, postData)
	}
}*/

func Integration() {
	variables.GlobalVar.SetIterator("driver_id", 3, 10)

	loop, err := strconv.Atoi(variables.GlobalVar.GetVar("loop"))
	if err != nil {
		loop = 1
	}

	for i := 0; i < loop; i++ {
		id := variables.GlobalVar.GetVar("driver_id")

		email := "moddie+driver" + id + "@HailoOSS.com"
		password := "Password1"
		device := "foobar"

		vars := li.Login(email, password, device)
		vars.UpdateVars(li.OnShift(vars.GetVar("driver_token")))
		li.OffShift(vars.GetVar("driver_token"), vars.GetVar("shift_id"))
		li.Logout(vars.GetVar("driver_token"))

		vars = li.LoginOnShift(email, password, device)
		li.OffShiftLogout(vars.GetVar("driver_token"), vars.GetVar("shift_id"))
	}
}
