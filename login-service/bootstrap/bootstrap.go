package main

import (
	"fmt"
	"os"
	"time"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/config/service_loader"
	"github.com/HailoOSS/service/sync"
	"github.com/HailoOSS/service/zookeeper"
)

const (
	username = "admin"
	password = "Password1"
)

func main() {
	sync.SetRegionLockNamespace("com.HailoOSS.service.login")

	fmt.Println("Loading config...")
	service_loader.Init("com.HailoOSS.service.login")
	_, t := config.LastLoaded()
	for t.IsZero() {
		fmt.Println("Config not loaded, sleeping for 0.5 seconds")
		time.Sleep(500 * time.Millisecond)

		_, t = config.LastLoaded()
		if !t.IsZero() {
			fmt.Println("Loaded config at", t.Format("2006-01-02 15:04:05"))
		}
	}
	if err := zookeeper.WaitForConnect(time.Second * 5); err != nil {
		fmt.Println("Failed to connect to ZK -- make sure you have H2 config service setup and running.")
		os.Exit(1)
	}
	fmt.Println("Loaded and connected to ZK.")

	u := &domain.User{
		App:     domain.Application("ADMIN"),
		Uid:     username,
		Ids:     []domain.Id{},
		Created: time.Now(),
		Roles:   []string{"ADMIN"},
	}
	if err := u.SetPassword(password); err != nil {
		fmt.Println("Error setting password: ", err)
		os.Exit(1)
	}

	if err := dao.CreateUser(u, password); err != nil {
		fmt.Println("Error creating user: ", err)
		os.Exit(1)
	}

	fmt.Println("Created user ", username, " ", password)
}
