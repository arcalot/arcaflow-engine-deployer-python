package tests

import (
	"encoding/json"
	"fmt"
	"go.arcalot.io/assert"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pythondeployer/internal/config"
	"go.flow.arcalot.io/pythondeployer/pkg/factory"
	"math/rand"
	"os"
	"os/exec"
	"testing"
)

type TestModule struct {
	Location string
	StepID   string
	Input    map[string]any
}

func GetPythonPath() (string, error) {
	python3Path, errPython3 := exec.LookPath("python3")
	if errPython3 != nil {
		pythonPath, errPython := exec.LookPath("python")
		if errPython != nil {
			return "", fmt.Errorf("error getting Python3 (%s) and python (%s)", errPython3, errPython)
		}
		return pythonPath, nil
	}
	return python3Path, nil
}

func CreateWorkdir(t *testing.T) string {
	workdir := fmt.Sprintf("/tmp/%s", RandString(10))
	if _, err := os.Stat(workdir); !os.IsNotExist(err) {
		err := os.RemoveAll(workdir)
		assert.NoError(t, err)
	}
	err := os.Mkdir(workdir, os.ModePerm)
	assert.NoError(t, err)
	return workdir
}

func RandString(n int) string {
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func GetConnector(t *testing.T, configJSON string, workdir *string) (deployer.Connector, *config.Config) {
	var serializedConfig any
	if err := json.Unmarshal([]byte(configJSON), &serializedConfig); err != nil {
		t.Fatal(err)
	}
	f := factory.NewFactory()
	schema := f.ConfigurationSchema()
	unserializedConfig, err := schema.UnserializeType(serializedConfig)
	assert.NoError(t, err)
	pythonPath, err := GetPythonPath()
	assert.NoError(t, err)
	unserializedConfig.PythonPath = pythonPath
	// NOTE: randomizing Workdir to avoid parallel tests to
	// remove python folders while other tests are running
	// causing the test to fail
	if workdir == nil {
		unserializedConfig.WorkDir = CreateWorkdir(t)
	} else {
		unserializedConfig.WorkDir = *workdir
	}

	connector, err := f.Create(unserializedConfig, log.NewTestLogger(t))
	assert.NoError(t, err)
	return connector, unserializedConfig
}
