package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	platformsync "github.com/hailo-platform/H2O/service/sync"

	sjson "github.com/bitly/go-simplejson"
)

const (
	sjsonnull = "null"
)

var (
	ErrPathNotFound   = errors.New("Config path not found")
	ErrIdNotFound     = errors.New("Config ID not found")
	DefaultRepository ConfigRepository

	emptyConfig = []byte("{}")
)

// ChangeSet represents some change to our config
type ChangeSet struct {
	// Id is a unique ID for the change
	Id string `cf:"configservice" key:"Id" name:"id" json:"id"`
	// Body is the JSON being applied
	Body []byte `name:"body" json:"body"`
	// Timestamp is when this happened
	Timestamp time.Time `name:"timestamp" json:"timestamp"`
	// UserMech identifies the authentication mechanism of the scope from which this change was applied
	UserMech string `name:"userMech" json:"userMech"`
	// UserId identifies the authenticated user ID of the scope from which this change was applied
	UserId string `name:"userId" json:"userId"`
	// Message tells us some human-readable message about the change
	Message string `name:"message" json:"message"`
	// ChangeId is some unique ID for this change
	ChangeId string `name:"changeId" json:"changeId"`
	// Path of the config being updated
	Path string `name:"path" json:"path"`
	// Old value for the config
	OldConfig []byte `name:"oldConfig" json:"oldConfig"`
}

type ConfigRepository interface {
	ReadConfig(ids []string) ([]*ChangeSet, error)
	UpdateConfig(cs *ChangeSet) error
	ChangeLog(start, end time.Time, count int, lastId string) ([]*ChangeSet, string, error)
	ServiceChangeLog(id string, start, end time.Time, count int, lastId string) ([]*ChangeSet, string, error)
}

func readConfigAtPath(body []byte, path string) ([]byte, error) {
	if len(path) == 0 {
		return body, nil
	}
	if len(body) == 0 {
		return nil, ErrPathNotFound
	}

	sj, err := sjson.NewJson(body)
	if err != nil {
		return nil, fmt.Errorf("Error parsing JSON: %v", err)
	}
	if path != "" {
		sj = sj.GetPath(strings.Split(path, "/")...)
	}

	bytes, err := sj.Encode()
	if err != nil {
		return nil, fmt.Errorf("Error getting bytes: %v", err)
	}
	if string(bytes) == sjsonnull {
		return nil, ErrPathNotFound
	}

	return bytes, nil
}

// ReadConfig returns the config item with the specified id.
// The path is optional, and if included will only return the contents at that path within the config.
// For reference, the changeset is included which will contain meta data about the last update
func ReadConfig(id, path string) ([]byte, *ChangeSet, error) {
	configs, err := DefaultRepository.ReadConfig([]string{id})
	if err != nil {
		return nil, nil, fmt.Errorf("Error getting config from DAO: %v", err)
	}
	if len(configs) != 1 {
		return nil, nil, ErrIdNotFound
	}

	b, err := readConfigAtPath(configs[0].Body, path)
	return b, configs[0], err
}

// ChangeLog returns a time series list of changes
func ChangeLog(start, end time.Time, count int, lastId string) ([]*ChangeSet, string, error) {
	chs, last, err := DefaultRepository.ChangeLog(start, end, count, lastId)
	return chs, last, err
}

// ChangeLog returns a time series list of changes for the given ID
func ServiceChangeLog(id string, start, end time.Time, count int, lastId string) ([]*ChangeSet, string, error) {
	chs, last, err := DefaultRepository.ServiceChangeLog(id, start, end, count, lastId)
	return chs, last, err
}

// mergeMap starts with "a" and recursivley adds "b" on top
// "a" will be modified
// bId is the id of "b" which is used for explaining
func mergeMap(a, b map[string]interface{}, bId string, stack []string, explain bool) []string {
	for k, v := range b {
		m, ok := v.(map[string]interface{})
		if ok {
			// Keep walking
			stack = append(stack, k)
			stack = mergeMap(a, m, bId, stack, explain)
		} else {
			// We're at a "leaf"
			// Walk down "a" creating nodes if we need to
			// and set the value
			anode := a
			for _, pos := range stack {
				if next, ok := anode[pos]; ok {
					anode = next.(map[string]interface{})
					continue
				}
				// Doesn't exist, create
				anode[pos] = make(map[string]interface{})
				anode = anode[pos].(map[string]interface{})
			}

			// Relace final node
			if explain {
				// In explain mode we set all values to the id
				// of the config they came from
				anode[k] = bId
			} else {
				anode[k] = v
			}
		}
	}
	// Pop item off stack
	if len(stack) > 0 {
		stack = stack[:len(stack)-1]
	}
	return stack
}

func compileConfig(ids []string, path string, explain bool) ([]byte, error) {
	if explain {
		// When explaining, we merge the first item in the list with itself
		// This ensures that to start with, all values appear to have come from
		// "a"
		ids = append(ids[:1], ids...)
	}

	configs, err := DefaultRepository.ReadConfig(ids)
	if err != nil {
		return nil, fmt.Errorf("Error getting configs: %v", err)
	}

	var compiled map[string]interface{}

	if len(configs) > 0 {
		err = json.Unmarshal(configs[0].Body, &compiled)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshalling config: %v", err)
		}

		// Merge!
		// Skip the first one since it's the base we're starting from
		for i := 1; i < len(configs); i++ {
			var config map[string]interface{}
			err := json.Unmarshal(configs[i].Body, &config)
			if err != nil {
				return nil, fmt.Errorf("Error unmarshalling config: %v", err)
			}
			mergeMap(compiled, config, configs[i].Id, make([]string, 0), explain)
		}
	}

	data, err := json.Marshal(compiled)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling compiled JSON: %v", err)
	}

	b, err := readConfigAtPath(data, path)
	if err == ErrPathNotFound {
		return emptyConfig, nil
	}
	return b, err
}

// CompileConfig will combine multiple configs together.
func CompileConfig(ids []string, path string) ([]byte, error) {
	return compileConfig(ids, path, false)
}

// ExplainConfig returns the compiled config except that instead of showing
// the original values, it shows which id was responsible for setting it
func ExplainConfig(ids []string, path string) ([]byte, error) {
	return compileConfig(ids, path, true)
}

// DeleteConfig will delete the node at the specified path
// It will return ErrPathNotFound if the path does not exist
func DeleteConfig(changeId, id, path, userMech, userId, message string) error {
	configs, err := DefaultRepository.ReadConfig([]string{id})
	if err != nil || len(configs) != 1 {
		return fmt.Errorf("Error getting config with id: %v", id)
	}

	var decoded map[string]interface{}
	err = json.Unmarshal(configs[0].Body, &decoded)
	if err != nil {
		return fmt.Errorf("Error decoding config: %v", err)
	}

	if path != "" {
		// Walk until we find the node we want
		// return early if we can't find it
		parts := strings.Split(path, "/")
		node := decoded
		ok := true
		for _, part := range parts[:len(parts)-1] {
			node, ok = node[part].(map[string]interface{})
			if !ok {
				return ErrPathNotFound
			}
		}
		delete(node, parts[len(parts)-1])
	} else {
		// No path, drop everything
		decoded = make(map[string]interface{})
	}

	encoded, err := json.Marshal(decoded)
	if err != nil {
		return fmt.Errorf("Error encoding config: %v", err)
	}

	oldConfig, err := readConfigAtPath(configs[0].Body, path)
	if err != nil {
		return fmt.Errorf("Error reading config at path %s : %s ", path, err.Error())
	}

	err = DefaultRepository.UpdateConfig(&ChangeSet{
		Id:        id,
		Body:      encoded,
		Timestamp: time.Now(),
		UserMech:  userMech,
		UserId:    userId,
		Message:   message,
		ChangeId:  changeId,
		Path:      path,
		OldConfig: oldConfig,
	})

	if err != nil {
		return fmt.Errorf("Error saving config: %v", err)
	}

	return nil
}

// CreateOrUpdateConfig will create or update the config for id at the specified path.
// Message should be a description of the change.
// Data should be the JSON data.
// userMech identifies the authentication mechanism of the scope from which this change was applied
func CreateOrUpdateConfig(changeId, id, path, userMech, userId, message string, data []byte) error {
	var newNode interface{}
	err := json.Unmarshal(data, &newNode)
	if err != nil {
		return fmt.Errorf("New value is not valid JSON: %v", err)
	}

	lock, err := platformsync.RegionLock([]byte(id))
	if err != nil {
		return err
	}
	defer lock.Unlock()

	configs, err := DefaultRepository.ReadConfig([]string{id})
	if err != nil {
		return fmt.Errorf("Error getting config from DAO: %v", err)
	}

	oldConfig := make([]byte, 0)
	if len(configs) == 1 {
		oldConfig, err = readConfigAtPath(configs[0].Body, path)

		if err == ErrPathNotFound {
			oldConfig = make([]byte, 0)
		} else if err != nil {
			return fmt.Errorf("Error getting config at path %s : %v", path, err)
		}
	}

	if path == "" {
		// If we are updating at the top level, it should be an object at the top level
		var target map[string]interface{}
		err = json.Unmarshal(data, &target)
		if err != nil {
			return fmt.Errorf("Top level config should be a JSON object")
		}

		return DefaultRepository.UpdateConfig(&ChangeSet{
			Id:        id,
			Body:      data,
			Timestamp: time.Now(),
			UserMech:  userMech,
			UserId:    userId,
			Message:   message,
			ChangeId:  changeId,
			Path:      path,
			OldConfig: oldConfig,
		})
	}

	decoded := make(map[string]interface{})
	if len(configs) == 1 && len(configs[0].Body) > 0 {
		err := json.Unmarshal(configs[0].Body, &decoded)
		if err != nil {
			return fmt.Errorf("Error parsing JSON: %v", err)
		}
	}

	// Walk down the path, making sure we have all the
	// parent nodes we need
	parent := decoded
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if i == len(parts)-1 {
			// Replace final node
			parent[part] = newNode
			break
		}

		node, ok := parent[part]
		if !ok {
			// Make new node
			parent[part] = make(map[string]interface{})
			parent = parent[part].(map[string]interface{})
			continue
		}

		parent = node.(map[string]interface{})
	}

	b, err := json.Marshal(decoded)
	if err != nil {
		return fmt.Errorf("Error encoding new config: %v", err)
	}

	return DefaultRepository.UpdateConfig(&ChangeSet{
		Id:        id,
		Body:      b,
		Timestamp: time.Now(),
		UserMech:  userMech,
		UserId:    userId,
		Message:   message,
		ChangeId:  changeId,
		Path:      path,
		OldConfig: oldConfig,
	})
}

func lockPath(id string) string {
	return fmt.Sprintf("/com.hailo-platform/H2O.service.config/%s", id)
}
