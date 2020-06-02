package loginintegration

import (
	"github.com/HailoOSS/hshell/integrationtest/request"
	"github.com/HailoOSS/hshell/integrationtest/validators"
	"github.com/HailoOSS/hshell/integrationtest/variables"
	"strconv"
	"time"
)

//Logs in a user
//Populates the 'driver_token' variable
func Login(email string, pass string, device string) *variables.TestVariables {
	vars := variables.NewVariables()
	postData := map[string]string{
		"email":    email,
		"password": pass,
		"device":   device,
	}
	rsp, _ := request.DoHttpRequest("Login",
		variables.GlobalVar.GetVar("driver_host"),
		"v1/driver/login",
		"POST",
		postData,
		validators.RegexValidator(`"status":true`),
	)
	vars.SetVarRegex(`"api_token":"([^"]+)"`, "driver_token", string(rsp))
	return vars
}

//Logs out a user
func Logout(apitoken string) {
	postData := map[string]string{
		"api_token": apitoken,
	}
	_, _ = request.DoHttpRequest("Logout",
		variables.GlobalVar.GetVar("driver_host"),
		"/v1/driver/logout",
		"POST",
		postData,
		validators.RegexValidator(`"status":true`),
	)
}

//Puts a driver on shift
//Populates the 'driver_token' variable
//Populates the 'shift_id' variable
func OnShift(apiToken string) *variables.TestVariables {
	vars := variables.NewVariables()
	postData := map[string]string{
		"api_token":      apiToken,
		"license_number": "12345",
		"timestamp":      strconv.Itoa(int(time.Now().Unix())),
	}
	rsp, _ := request.DoHttpRequest("Shift Start",
		variables.GlobalVar.GetVar("driver_host"),
		"/v1/shift/start",
		"POST",
		postData,
		validators.RegexValidator(`"status":true`),
	)
	vars.SetVarRegex(`"api_token":"([^"]+)"`, "driver_token", string(rsp))
	vars.SetVarRegex(`"shift":"([^"]+)"`, "shift_id", string(rsp))
	return vars
}

//Puts a driver off shift
func OffShift(apiToken string, shiftId string) {
	postData := map[string]string{
		"api_token": apiToken,
		"shift":     shiftId,
		"timestamp": strconv.Itoa(int(time.Now().Unix())),
	}

	_, _ = request.DoHttpRequest("Shift End",
		variables.GlobalVar.GetVar("driver_host"),
		"/v1/shift/end",
		"POST",
		postData,
		validators.RegexValidator(`"status":true`),
	)
}

// Logs in and Puts a driver on shift
//Populates the 'driver_token' variable
//Populates the 'shift_id' variable
func LoginOnShift(email string, pass string, device string) *variables.TestVariables {
	vars := Login(email, pass, device)
	vars.UpdateVars(OnShift(vars.GetVar("driver_token")))
	return vars
}

//Takes the driver offshift and logs out.
func OffShiftLogout(apiToken string, shiftId string) {
	OffShift(apiToken, shiftId)
	Logout(apiToken)
}
