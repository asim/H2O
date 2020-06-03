package domain

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	platformtesting "github.com/hailo-platform/H2O/platform/testing"
	ssync "github.com/hailo-platform/H2O/service/sync"
	zk "github.com/hailo-platform/H2O/service/zookeeper"
	gozk "github.com/hailo-platform/H2O/go-zookeeper/zk"
)

type DomainSuite struct {
	platformtesting.Suite
	zk *zk.MockZookeeperClient
}

func TestRunDomainSuite(t *testing.T) {
	platformtesting.RunSuite(t, new(DomainSuite))
}

func (s *DomainSuite) SetupTest() {
	s.Suite.SetupTest()
	s.zk = &zk.MockZookeeperClient{}
	zk.ActiveMockZookeeperClient = s.zk
	zk.Connector = zk.MockConnector
	ssync.SetRegionLockNamespace("com.hailo-platform/H2O.service.config")
}

func (s *DomainSuite) TearDownTest() {
	s.Suite.TearDownTest()
	zk.ActiveMockZookeeperClient = nil
	zk.Connector = zk.DefaultConnector
}

// Sample JSON taken from the old config service
var sampleJson = `{
  "hailo": {
    "failureRhino": {
      "webservice": {
        "frontController": {
          "failureChance": 0,
          "latencyChance": 0,
          "latencyRange": "200|1000"
        }
      }
    }
  }
}`

func compareJson(a, b []byte) (bool, error) {
	var ja, jb interface{}
	err := json.Unmarshal(a, &ja)
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(b, &jb)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(ja, jb), nil
}

func (s *DomainSuite) TestGetConfig() {
	testRepo := &memoryRepository{
		data: map[string]*ChangeSet{
			"services": &ChangeSet{
				Id:        "services",
				Body:      []byte(sampleJson),
				Timestamp: time.Now(),
			},
		},
	}

	DefaultRepository = testRepo

	testCases := []struct {
		key      string
		path     string
		expected string
		err      error
	}{
		{
			"services",
			"",
			sampleJson,
			nil,
		},
		{
			"services",
			"hailo/failureRhino/webservice",
			`{
        "frontController": {
          "failureChance": 0,
          "latencyChance": 0,
          "latencyRange": "200|1000"
        }
      }`,
			nil,
		},
		{
			"services",
			"hailo/not_there",
			`null`,
			ErrPathNotFound,
		},
	}

	for i, tc := range testCases {
		result, _, err := ReadConfig(tc.key, tc.path)
		if tc.err != nil {
			s.Equal(tc.err, err)
			continue
		}
		s.NoError(err)

		eq, err := compareJson([]byte(tc.expected), result)
		s.NoError(err)
		s.True(eq, "Expected JSON does not match for testcase %v", i)
	}
}

func (s *DomainSuite) TestUpdateConfig() {
	testCases := []struct {
		initialData string
		key         string
		path        string
		newValue    string
		expected    string
		shouldError bool
	}{
		// Starting with no data
		{``, "test", "", "{}", "{}", false},
		// Top level items should be an object
		{"", "test", "", `[{}]`, ``, true},
		// Top level items should be an object
		{"", "test", "", `\"\"`, ``, true},
		// Allow updating values at the top level
		{"", "test", "currency", `"USD"`, `{"currency":"USD"}`, false},
		// Update without path
		{sampleJson, "services", "", "{}", "{}", false},
		// Update with path
		{`{
  "hailo": {
    "failureRhino": {
      "webservice": {
        "frontController": {
          "failureChance": 0,
          "latencyChance": 0,
          "latencyRange": "200|1000"
        }
      }}}}`,
			"services",
			"hailo/failureRhino/webservice",
			`{"frontController": {
          "failureChance": 1,
          "latencyChance": 1,
          "latencyRange": ""
        }}`, `{
  "hailo": {
    "failureRhino": {
      "webservice": {
        "frontController": {
          "failureChance": 1,
          "latencyChance": 1,
          "latencyRange": ""
        }
      }}}}`, false},
		// Update single value
		{
			`{
  "hailo": {
    "failureRhino": {
      "webservice": {
        "frontController": {
          "failureChance": 0,
          "latencyChance": 0,
          "latencyRange": "200|1000"
        }
      }}}}`,
			"services",
			"hailo/failureRhino/webservice/frontController/failureChance",
			"1",
			`{
  "hailo": {
    "failureRhino": {
      "webservice": {
        "frontController": {
          "failureChance": 1,
          "latencyChance": 0,
          "latencyRange": "200|1000"
        }
      }}}}`, false,
		},
		// Add value
		{`{
  "hailo": {
    "failureRhino": {
      "webservice": {
        "frontController": {
          "failureChance": 0,
          "latencyChance": 0,
          "latencyRange": "200|1000"
        }
      }}}}`,
			"services",
			"hailo/failureRhino/a/b",
			`{
        "frontController": {
          "failureChance": 1,
          "latencyChance": 1,
          "latencyRange": ""
        }}`,
			`{
  "hailo": {
    "failureRhino": {
      "webservice": {
        "frontController": {
          "failureChance": 0,
          "latencyChance": 0,
          "latencyRange": "200|1000"
        }
      },
      "a": {
        "b": {
          "frontController": {
            "failureChance": 1,
            "latencyChance": 1,
            "latencyRange": ""
          }
        }
      }
    }
  }
}`, false,
		},
	}

	for i, tc := range testCases {
		testRepo := &memoryRepository{
			data: map[string]*ChangeSet{
				tc.key: &ChangeSet{
					Id:        tc.key,
					Body:      []byte(tc.initialData),
					Timestamp: time.Now(),
				},
			},
		}

		DefaultRepository = testRepo

		s.zk.
			On("NewLock", lockPath(tc.key), gozk.WorldACL(gozk.PermAll)).
			Return(&mockLock{})

		err := CreateOrUpdateConfig("foo", tc.key, tc.path, "Test Message", "h2", "dave", []byte(tc.newValue))
		if !tc.shouldError {
			s.NoError(err)
			continue
		}
		if tc.shouldError {
			s.Error(err)
			continue
		}

		data := testRepo.data[tc.key]
		eq, err := compareJson([]byte(tc.expected), data.Body)
		s.NoError(err, "Error comparing JSON: %v (%v)", err, i)
		s.True(eq,
			"New value incorrect. Expected: \n%v\nGot: %v",
			string(tc.expected),
			string(data.Body))
	}
}

var (
	compileA = `{
  "hailo": {
    "service1": {
      "value1": 10,
      "value2": 20
    },
    "service2": {
      "value1": 10,
      "hosts": [
        "a",
        "b",
        "c"
      ]
    }
  }
}`

	compileB = `{
  "hailo": {
    "service2": {
      "value1": 20,
      "hosts": [
        "d"
      ],
      "value3": 30
    },
    "service3": {
      "value1": 10
    }
  }
}`

	compileC = `{
  "hailo": {
    "service1": {
      "value1": 10,
      "value2": 20
    },
    "service2": {
      "value1": 20,
      "hosts": [
        "d"
      ],
      "value3": 30
    },
    "service3": {
      "value1": 10
    }
  }
}`

	explainC = `{
  "hailo": {
    "service1": {
      "value1": "a",
      "value2": "a"
    },
    "service2": {
      "value1": "b",
      "hosts":  "b",
      "value3": "b"
    },
    "service3": {
      "value1": "b"
    }
  }
}`
)

func (s *DomainSuite) TestCompileConfig() {
	testCases := []struct {
		keys     []string
		path     string
		expected string
	}{
		// Test that compiling "b" onto "a" gives us "c"
		{[]string{"a", "b"}, "", compileC},
		// Same with path
		{[]string{"a", "b"}, "hailo/service2", `{
      "value1": 20,
      "hosts": [
        "d"
      ],
      "value3": 30
    }`},
	}

	for _, tc := range testCases {
		testRepo := &memoryRepository{
			data: map[string]*ChangeSet{
				"a": &ChangeSet{
					Id:        "a",
					Body:      []byte(compileA),
					Timestamp: time.Now(),
				},
				"b": &ChangeSet{
					Id:        "b",
					Body:      []byte(compileB),
					Timestamp: time.Now(),
				},
			},
		}

		DefaultRepository = testRepo

		compiled, err := CompileConfig(tc.keys, tc.path)
		s.NoError(err)

		eq, err := compareJson([]byte(tc.expected), compiled)
		s.NoError(err, "Error comparing JSON")
		s.True(eq,
			"New value incorrect.\nExpected:\n%v\nGot:\n%v",
			string(tc.expected),
			string(compiled))
	}
}

func (s *DomainSuite) TestExplainConfig() {
	// Test that compiling "b" onto "a" gives us "c"
	testRepo := &memoryRepository{
		data: map[string]*ChangeSet{
			"a": &ChangeSet{
				Id:        "a",
				Body:      []byte(compileA),
				Timestamp: time.Now(),
			},
			"b": &ChangeSet{
				Id:        "b",
				Body:      []byte(compileB),
				Timestamp: time.Now(),
			},
		},
	}

	DefaultRepository = testRepo

	explained, err := ExplainConfig([]string{"a", "b"}, "")
	s.NoError(err)

	eq, err := compareJson([]byte(explainC), explained)
	s.NoError(err, "Error comparing JSON")
	s.True(eq,
		"New value incorrect.\nExpected:\n%v\nGot:\n%v",
		string(explainC),
		string(explained))
}

func (s *DomainSuite) TestDeleteConfig() {
	initialData := `{
  "hailo": {
    "service1": {
      "value1": 10,
      "value2": 20
    },
    "service2": {
      "value1": 10,
      "hosts": [
        "a",
        "b",
        "c"
      ]
    }
  }
}`

	testCases := []struct {
		key      string
		path     string
		expected string
	}{
		{"a", "hailo/service2", `{
  "hailo": {
    "service1": {
      "value1": 10,
      "value2": 20
    }
  }
}`},
		{"a", "", `{}`},
		{"a", "hailo/service2/value1", `{
  "hailo": {
    "service1": {
      "value1": 10,
      "value2": 20
    },
    "service2": {
      "hosts": [
        "a",
        "b",
        "c"
      ]
    }
  }
}`},
	}

	for _, tc := range testCases {
		testRepo := &memoryRepository{
			data: map[string]*ChangeSet{
				"a": &ChangeSet{
					Id:        "a",
					Body:      []byte(initialData),
					Timestamp: time.Now(),
				},
			},
		}

		DefaultRepository = testRepo

		err := DeleteConfig("foo", tc.key, tc.path, "h2", "dave", "Test Message")
		s.NoError(err)

		updated := testRepo.data["a"]
		eq, err := compareJson([]byte(tc.expected), updated.Body)
		s.NoError(err, "Error comparing JSON")

		s.True(eq,
			"New value incorrect.\nExpected:\n%v\nGot:\n%v",
			string(tc.expected),
			string(updated.Body))
	}

}

// Verifies concurrent updates do not overwrite eachother
func (s *DomainSuite) TestUpdateConfigConcurrently() {
	id := "test"
	userMech := "h2"
	userId := "dave"
	initialData := `{ "a": 0, "b": 0 }`

	testRepo := &memoryRepository{
		data: map[string]*ChangeSet{
			id: &ChangeSet{
				Id:        id,
				Body:      []byte(initialData),
				Timestamp: time.Now(),
			},
		},
	}

	DefaultRepository = testRepo

	lock := &mockLock{}
	s.zk.
		On("NewLock", lockPath(id), gozk.WorldACL(gozk.PermAll)).
		Return(lock)

	paths := []string{"a", "b"}

	for i := 1; i <= 100; i++ {
		wg := sync.WaitGroup{}
		wg.Add(len(paths))

		for _, path := range paths {
			go func(path string) {
				msg := fmt.Sprintf("Setting %s to %d", path, i)
				err := CreateOrUpdateConfig(fmt.Sprintf("%s:%d", path, i), id, path, userMech, userId, msg, []byte(fmt.Sprintf("%d", i)))
				s.NoError(err)

				wg.Done()
			}(path)
		}

		wg.Wait()
		data, _, err := ReadConfig(id, "")
		s.NoError(err, "Failed to update config")

		config := map[string]int{}
		err = json.Unmarshal(data, &config)
		s.NoError(err, "Failed to unmarshal config")

		for _, path := range paths {
			s.Equal(config[path], i)
		}
	}
}

// mockLock is used for test purposes
type mockLock struct {
	m sync.Mutex
}

func (l *mockLock) Lock() error {
	l.m.Lock()
	return nil
}

func (l *mockLock) Unlock() error {
	l.m.Unlock()
	return nil
}

func (l *mockLock) SetTTL(x time.Duration)     {}
func (l *mockLock) SetTimeout(x time.Duration) {}
