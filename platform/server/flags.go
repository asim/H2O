package server

import (
	"flag"
	"fmt"
	"os"
)

var (
	flagPrintInfo    bool
	flagPrintName    bool
	flagPrintVersion bool
)

func init() {
	setupFlags()
}

func setupFlags() {
	flag.BoolVar(&flagPrintInfo, "i", false, "Print service information and exit.")
	flag.BoolVar(&flagPrintName, "name", false, "Print service name and exit.")
	flag.BoolVar(&flagPrintVersion, "version", false, "Print service version and exit.")
}

func handleFlags() {
	flag.Parse()

	if flagPrintInfo {
		fmt.Println("Name:        ", Name)
		fmt.Println("Description: ", Description)
		fmt.Println("Version:     ", Version)
		fmt.Println("Source:      ", Source)
		fmt.Println("OwnerEmail:  ", OwnerEmail)
		fmt.Println("OwnerMobile: ", OwnerMobile)
		fmt.Println("OwnerTeam:   ", OwnerTeam)
		os.Exit(0)
	}

	if flagPrintVersion {
		fmt.Println(Version)
		os.Exit(0)
	}

	if flagPrintName {
		fmt.Println(Name)
		os.Exit(0)
	}
}
