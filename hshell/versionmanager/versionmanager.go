package versionmanager

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/HailoOSS/hshell/provision"
	"io/ioutil"
	"strconv"
)

const (
	ext = ".versions"
)

func ExportVersions(filename string) error {
	vers, err := provision.GetServiceVersionMachineClass()
	if err != nil {
		return err
	}
	toWrite := formatForFile(vers)

	// write whole the body
	err = ioutil.WriteFile(filename, toWrite, 0644)
	if err != nil {
		return err
	}
	return nil
}

func ImportVersions(filename string) error {
	// read whole the file
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	vers, err := unmarshalFile(b)
	if err != nil {
		return err
	}
	return UpgradeServices(vers)

}

func formatForFile(ver []*provision.ServiceVersionMachine) []byte {

	toPrint := ""

	for _, val := range ver {
		fmt.Printf("%s,%s,%v\n", val.Service, val.MachineClass, val.Version)
		toPrint += fmt.Sprintf("%s,%s,%v\n", val.Service, val.MachineClass, val.Version)
	}

	return []byte(toPrint)
}

func unmarshalFile(b []byte) ([]*provision.ServiceVersionMachine, error) {
	reader := csv.NewReader(bytes.NewReader(b))

	fields, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	svm := make([]*provision.ServiceVersionMachine, len(fields))
	for i, field := range fields {
		if len(field) == 3 {
			svm[i] = &provision.ServiceVersionMachine{}
			svm[i].Service = field[0]
			svm[i].MachineClass = field[1]
			svm[i].Version, _ = strconv.Atoi(field[2])
		}
	}
	return svm, nil
}

func UpgradeServices(vers []*provision.ServiceVersionMachine) error {

	versOld, err := provision.GetServiceVersionMachineClass()
	if err != nil {
		return err
	}

	for _, ver := range vers {
		err := provision.CreateService(ver.Service, ver.MachineClass, ver.Version)
		if err != nil {
			fmt.Println("could not provision service ", ver)
			continue
		}

		for _, verOld := range versOld {
			if verOld.Service == ver.Service && verOld.Version != ver.Version {
				err := provision.DeleteService(verOld.Service, verOld.MachineClass, verOld.Version)
				if err != nil {
					fmt.Println("could not remove old service ", verOld)
				}
			}

		}

	}
	return nil
}
