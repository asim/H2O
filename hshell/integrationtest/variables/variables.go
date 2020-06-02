package variables

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"
)

type TestVariables struct {
	variables map[string]Variable
	sync.RWMutex
}

type Variable interface {
	String() string
	SetString(string)
}

type StringVar struct {
	string
}

func (sv *StringVar) String() string {
	return sv.string
}

func (sv *StringVar) SetString(s string) {
	sv.string = s
}

type Iterator struct {
	int
	min int
	max int
}

func (i *Iterator) String() string {
	val := strconv.Itoa(i.int)
	i.SetString(strconv.Itoa(i.int + 1))
	return val
}

func (i *Iterator) SetString(s string) {
	val, _ := strconv.Atoi(s)
	if val > i.max || val < i.min {
		i.int = i.min
	}
	i.int = val
}

var GlobalVar *TestVariables = NewVariables()

func NewVariables() *TestVariables {
	return &TestVariables{variables: make(map[string]Variable)}
}

func (tv *TestVariables) GetVar(name string) string {
	tv.RLock()
	defer tv.RUnlock()
	if _, ok := tv.variables[name]; !ok {
		return ""
	}
	return tv.variables[name].String()
}

func (tv *TestVariables) SetVar(name string, value string) {
	tv.Lock()
	defer tv.Unlock()
	if _, ok := tv.variables[name]; !ok {
		tv.variables[name] = &StringVar{}
	}
	tv.variables[name].SetString(value)
}

func (tv *TestVariables) SetVarRegex(regex string, name string, s string) {
	grabbed := regexp.MustCompile(regex).FindStringSubmatch(s)
	if grabbed != nil && len(grabbed) == 2 {
		tv.SetVar(name, grabbed[1])
	}
}

func (tv *TestVariables) UpdateVars(vars *TestVariables) {
	for name, val := range vars.variables {
		tv.SetVar(name, val.String())
	}
}

func (tv *TestVariables) SetIterator(name string, min int, max int) error {
	tv.Lock()
	defer tv.Unlock()

	if _, ok := tv.variables[name]; ok {
		return fmt.Errorf("Variable already exists")
	}

	iter := &Iterator{min: min, max: max}
	iter.SetString(strconv.Itoa(min))
	tv.variables[name] = iter

	return nil
}

func (tv *TestVariables) GetAvailableVariables() []string {
	tv.RLock()
	defer tv.RUnlock()
	s := make([]string, len(tv.variables))
	i := 0
	for name, _ := range tv.variables {
		s[i] = name
	}
	return s
}
