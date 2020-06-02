package hintegrate

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	auth "github.com/HailoOSS/login-service/proto/auth"
	"github.com/HailoOSS/hintegrate/request"
	"github.com/HailoOSS/hintegrate/validators"
)

const (
	adminUserFallback = "admin"
	adminPassFallback = "Password1"
	mech              = "h2"
	application       = "ADMIN"
	serviceName       = "com.HailoOSS.service.login"
)

// Stores found admin tokens in a map based on callapi_host
// this is a proxyId for the environment this token works for
// I should expire these tokens at some point.  Probably just
// do a read and see if it is still valid when I assign.
var (
	adminToken map[string]string = make(map[string]string)
	tokenLock  sync.RWMutex
)

// GetAdminToken retursn an admin token and sets it to the given context
func GetAdminToken(c *request.Context) (string, error) {
	tokenLock.Lock()
	defer tokenLock.Unlock()

	for _, host := range c.Hosts["callapi_host"] {
		if len(adminToken[host]) != 0 {

			_, err := ReadSession(c, adminToken[host], false)
			if err != nil {
				break
			}
			c.Vars.SetVar("admin_token", adminToken[host])
			return adminToken[host], nil
		}
	}

	session, err := adminLogin(c, c.Vars.GetVar("admin_user"), c.Vars.GetVar("admin_pass"))

	if err != nil {
		session, err = adminLogin(c, adminUserFallback, adminPassFallback)
	}

	for _, host := range c.Hosts["callapi_host"] {
		adminToken[host] = session
	}

	return session, err
}

// adminLogin gets an admin token for the hardcoded admin user (hintegrate)
func adminLogin(c *request.Context, user, pass string) (string, error) {

	rand.Seed(int64(time.Now().Nanosecond()))

	stringMap := make(map[string]string)
	stringMap["Username"] = user
	stringMap["Password"] = pass
	stringMap["Application"] = application
	stringMap["Mech"] = mech
	stringMap["DeviceType"] = strconv.Itoa(rand.Intn(1000000))

	adminJson, err := json.Marshal(stringMap)
	if err != nil {
		return "", err
	}
	postData := map[string]string{
		"service":  "com.HailoOSS.service.login",
		"endpoint": "auth",
		"request":  string(adminJson),
	}

	rsp, err := c.Post().SetHost("callapi_host").
		PostDataMap(postData).SetPath("/rpc").
		Run(serviceName+".auth", validators.Status2xxValidator())

	if err != nil {
		return "", err
	}
	authRsp := &auth.Response{}
	err = json.Unmarshal(rsp.Body, authRsp)
	if err != nil {
		fmt.Println(string(rsp.Body))
		return "", err
	}

	return authRsp.GetSessId(), nil
}
