package main

import (
	"flag"
	"fmt"
	"os"

	cfg "github.com/HailoOSS/config-service/config"
	"github.com/HailoOSS/config-service/dao"
	"github.com/HailoOSS/config-service/domain"
)

var (
	id      = flag.String("id", "H2:BASE", "The ID of config to update")
	config  = flag.String("config", "", "Configuration JSON")
	message = flag.String("message", "", "Commit message for this change")
)

func main() {
	flag.Parse()
	cfg.Bootstrap()
	domain.DefaultRepository = &dao.CassandraRepository{}

	err := domain.CreateOrUpdateConfig(
		"init",
		*id,
		"",
		"none",
		"none",
		*message,
		[]byte(*config),
	)
	if err != nil {
		fmt.Println("Failed to bootstrap config: ", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully bootstrapped config for ID '%v': %v", *id, *config)
}
