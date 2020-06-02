package importstate

import (
	"fmt"
	"github.com/HailoOSS/hshell/integrationtest/variables"
	"github.com/jimlawless/cfg"
)

func GetVarsFromFile(file string) error {
	confMap := make(map[string]string)
	err := cfg.Load(file, confMap)
	if err != nil {
		return fmt.Errorf("Can't parse config file: %e", err)
	}

	for name, value := range confMap {
		variables.GlobalVar.SetVar(name, value)
	}
	return nil
}
