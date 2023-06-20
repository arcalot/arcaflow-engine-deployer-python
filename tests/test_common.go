package tests

import (
	"encoding/json"
	"fmt"
	pythondeployer "github.com/arcalot/arcaflow-engine-deployer-python"
	wrapper "github.com/arcalot/arcaflow-engine-deployer-python/internal/cliwrapper"
	"github.com/arcalot/arcaflow-engine-deployer-python/internal/config"
	"go.arcalot.io/assert"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"math/rand"
	"os"
	"testing"
)

func createWorkdir(t *testing.T) string {
	workdir := fmt.Sprintf("/tmp/%s", randString(10))
	if _, err := os.Stat(workdir); !os.IsNotExist(err) {
		err := os.RemoveAll(workdir)
		assert.NoError(t, err)
	}
	err := os.Mkdir(workdir, os.ModePerm)
	assert.NoError(t, err)
	return workdir
}

func randString(n int) string {
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func removeModuleIfExists(module string, python wrapper.CliWrapper, t *testing.T) {
	modulePath, err := python.GetModulePath(module)
	assert.Nil(t, err)
	if _, err := os.Stat(*modulePath); !os.IsNotExist(err) {
		os.RemoveAll(*modulePath)
	}
}

func pullModule(python wrapper.CliWrapper, module string, workDir string, t *testing.T) error {
	removeModuleIfExists(module, python, t)
	err := python.PullModule(module)
	if err != nil {
		return err
	}
	return nil
}

func getCliWrapper(t *testing.T, workdir string) wrapper.CliWrapper {
	workDir := workdir
	pythonPath := "/usr/bin/python3.9"
	logger := log.NewTestLogger(t)
	return wrapper.NewCliWrapper(pythonPath, workDir, logger)
}

func getConnector(t *testing.T, configJSON string, workdir *string) (deployer.Connector, *config.Config) {
	var serializedConfig any
	if err := json.Unmarshal([]byte(configJSON), &serializedConfig); err != nil {
		t.Fatal(err)
	}
	factory := pythondeployer.NewFactory()
	schema := factory.ConfigurationSchema()
	unserializedConfig, err := schema.UnserializeType(serializedConfig)
	assert.NoError(t, err)
	// NOTE: randomizing Workdir to avoid parallel tests to
	// remove python folders while other tests are running
	// causing the test to fail
	if workdir == nil {
		unserializedConfig.WorkDir = createWorkdir(t)
	} else {
		unserializedConfig.WorkDir = *workdir
	}

	connector, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
	assert.NoError(t, err)
	return connector, unserializedConfig
}
