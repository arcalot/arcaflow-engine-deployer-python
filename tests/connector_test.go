package tests

import (
	"context"
	"fmt"
	"go.arcalot.io/assert"
	"go.arcalot.io/log/v2"
	"go.flow.arcalot.io/deployer"
	"go.flow.arcalot.io/pluginsdk/atp"
	"go.flow.arcalot.io/pluginsdk/schema"
	"go.flow.arcalot.io/pythondeployer"
	"os"
	"sync"
	"testing"
)

const examplePluginNickname string = "pythonuser"

var inOutConfigGitPullIfNotPresent = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"IfNotPresent",
    "semver": "3.0.0"
}
`

func TestRunStepGit(t *testing.T) {
	moduleName := "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git@52d1a9559c60a615dbd97114572f16d70fa30b1b"
	stepID := "hello-world"
	input := map[string]any{
		"name": examplePluginNickname,
	}

	connector, _ := getConnector(t, inOutConfigGitPullIfNotPresent, nil)
	OutputID, OutputData, Error := RunStep(t, connector, moduleName, stepID, input)
	assert.NoError(t, Error)
	assert.Equals(t, OutputID, "success")
	assert.Equals(t,
		OutputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", examplePluginNickname)})
}

func RunStep(t *testing.T, connector deployer.Connector, moduleName string, stepID string, input map[string]any) (string, any, error) {
	plugin, err := connector.Deploy(context.Background(), moduleName)

	if err != nil {
		return "", nil, err
	}
	t.Cleanup(func() {
		assert.NoError(t, plugin.Close())
	})

	atpClient := atp.NewClient(plugin)
	pluginSchema, err := atpClient.ReadSchema()
	assert.NoError(t, err)
	assert.NotNil(t, pluginSchema)
	steps := pluginSchema.Steps()
	step, ok := steps[stepID]
	if !ok {
		t.Fatalf("no such step: %s", stepID)
	}

	_, err = step.Input().Unserialize(input)
	assert.NoError(t, err)

	// Executes the step and validates that the output is correct.
	receivedSignalsChan := make(chan schema.Input)
	emittedSignalsChan := make(chan schema.Input)
	executionResult := atpClient.Execute(
		schema.Input{RunID: t.Name(), ID: stepID, InputData: input},
		receivedSignalsChan,
		emittedSignalsChan,
	)
	assert.NoError(t, atpClient.Close())

	return executionResult.OutputID, executionResult.OutputData, executionResult.Error
}

// This test ensures that this deployer can create and execute
// connectors concurrently, and that those connectors can deploy
// plugins concurrently that create side-effects local to their
// filesystem, and one connector can pull multiple python modules
func TestDeployConcurrent_ConnectorsAndPluginsWithDifferentModules(t *testing.T) {
	type TestModule struct {
		location string
		stepID   string
		input    map[string]any
	}
	testModules := map[string]TestModule{
		"fio": {
			stepID:   "workload",
			location: "arcaflow-plugin-fio@git+https://github.com/arcalot/arcaflow-plugin-fio.git@de07b3e48cefdaa084eb0445616abc2d13670191",
			input: map[string]any{
				"name":    "poisson-rate-submit",
				"cleanup": "true",
				"params": map[string]any{
					"size":           "90KiB",
					"readwrite":      "randrw",
					"ioengine":       "sync",
					"iodepth":        32,
					"io_submit_mode": "inline",
					"rate_iops":      50,
					"rate_process":   "poisson",
					"buffered":       0,
				},
			},
		},
		"template": {
			stepID:   "hello-world",
			location: "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git@52d1a9559c60a615dbd97114572f16d70fa30b1b",
			input: map[string]any{
				"name": "arca lot",
			},
		},
		"wait": {
			stepID:   "wait",
			location: "arcaflow-plugin-wait@git+https://github.com/arcalot/arcaflow-plugin-wait.git",
			input: map[string]any{
				"seconds": "0.05",
			},
		},
	}

	rootDir := "/tmp/multi-module"
	serializedConfig := map[string]any{
		"workdir":          rootDir,
		"modulePullPolicy": "IfNotPresent",
		"semver":           "3.0.0",
	}

	assert.NoError(t, os.MkdirAll(rootDir, os.ModePerm))

	factory := pythondeployer.NewFactory()
	deployerSchema := factory.ConfigurationSchema()
	unserializedConfig, err := deployerSchema.UnserializeType(serializedConfig)
	assert.NoError(t, err)

	pythonPath, err := getPythonPath()
	assert.NoError(t, err)
	unserializedConfig.PythonPath = pythonPath

	// Choose how many connectors and plugins to make
	const n_connectors = 4
	const n_plugin_copies = 3
	wg := &sync.WaitGroup{}
	wg.Add(n_connectors * len(testModules) * n_plugin_copies)

	logger := log.NewTestLogger(t)

	// Test for issues that might occur during concurrent creation of connectors
	// and deployment of plugins
	// Make a goroutine for each connector
	for j := 0; j < n_connectors; j++ {
		connector_, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
		assert.NoError(t, err)

		go func(connector deployer.Connector) {
			for k := 0; k < n_plugin_copies; k++ {
				for _, testModule_ := range testModules {
					go func(testModule TestModule) {
						defer wg.Done()

						output_id, output_data, err := RunStep(
							t, connector, testModule.location, testModule.stepID, testModule.input)
						assert.NoError(t, err)
						assert.Equals(t, output_id, "success")
						assert.MapNotContainsKeyAny(t, "error", output_data.(map[any]any))
						if output_id == "error" {
							errorMsg, ok := output_data.(map[any]any)["error"]
							if ok {
								logger.Debugf("plugin error '%s'", errorMsg.(string))
							}
						}
					}(testModule_)
				}
			}
		}(connector_)
	}
	// Wait for all the plugins to be done
	wg.Wait()

	//t.Cleanup(func() {
	//	assert.NoError(t, os.RemoveAll(rootDir))
	//})
}
