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
	"modulePullPolicy":"IfNotPresent"
}
`

func TestRunStepGit(t *testing.T) {
	//moduleName := "arcaflow-plugin-example@git+https://github.com/arcalot/arcaflow-plugin-example.git"
	moduleName := "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git@1dd9a772e65397f5f56af3b907f84fcdfbd9ff28"
	connector, _ := getConnector(t, inOutConfigGitPullIfNotPresent, nil)
	stepID := "hello-world"
	input := map[string]any{
		"name": examplePluginNickname,
	}
	OutputID, OutputData, Error := RunStep(t, connector, moduleName, stepID, input)
	assert.NoError(t, Error)
	assert.Equals(t, OutputID, "success")
	assert.Equals(t,
		OutputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", examplePluginNickname)})
}

func RunStep(t *testing.T, connector deployer.Connector, moduleName string, stepID string, input map[string]any) (string, any, error) {
	//stepID := "hello-world"
	//input := map[string]any{
	//	"name": map[string]any{
	//		"_type": "nickname",
	//		"nick":  examplePluginNickname,
	//	},
	//}

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
// plugins concurrently.
func TestDeployConcurrent_ConnectorsAndPlugins(t *testing.T) {
	moduleName := "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git@4fd43570277995f89a5f17fe4beb4374edf44d68"
	rootDir := "/tmp/template"
	serializedConfig := map[string]any{
		"workdir":          rootDir,
		"modulePullPolicy": "Always",
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
	const n_connectors = 3
	const n_plugins = 3
	wg := &sync.WaitGroup{}
	wg.Add(n_connectors * n_plugins)

	// Test for issues that might occur during concurrent creation of connectors
	// and deployment of plugins
	// Make a goroutine for each connector
	for j := 0; j < n_connectors; j++ {
		go func() {
			connector, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
			assert.NoError(t, err)

			// Make a goroutine for each plugin
			for k := 0; k < n_plugins; k++ {
				go func() {
					plugin, err := connector.Deploy(context.Background(), moduleName)
					assert.NoError(t, err)
					assert.NoError(t, plugin.Close())
					wg.Done()
				}()
			}
		}()
	}
	// Wait for all the plugins to be done
	wg.Wait()

	assert.NoError(t, os.Remove(rootDir))
}

// This test ensures that this deployer can create and execute
// connectors concurrently, and that those connectors can deploy
// plugins concurrently that create side-effects local to their
// filesystem
func TestDeployConcurrent_ConnectorsAndPluginsWithFilesystemSideEffects(t *testing.T) {

	moduleName := "arcaflow-plugin-fio@git+https://github.com/arcalot/arcaflow-plugin-fio.git@122f26195130d51cccb3db142c6ae8f783bac57c"
	rootDir := "/tmp/fio"
	serializedConfig := map[string]any{
		"workdir":          rootDir,
		"modulePullPolicy": "IfNotPresent",
	}

	assert.NoError(t, os.MkdirAll(rootDir, os.ModePerm))

	stepID := "workload"
	input := map[string]any{
		"name":    "poisson-rate-submit",
		"cleanup": "true",
		"params": map[string]any{
			"size":           "100KiB",
			"readwrite":      "randrw",
			"ioengine":       "sync",
			"iodepth":        32,
			"io_submit_mode": "inline",
			"rate_iops":      50,
			"rate_process":   "poisson",
			"buffered":       0,
		},
	}

	factory := pythondeployer.NewFactory()
	deployerSchema := factory.ConfigurationSchema()
	unserializedConfig, err := deployerSchema.UnserializeType(serializedConfig)
	assert.NoError(t, err)

	pythonPath, err := getPythonPath()
	assert.NoError(t, err)
	unserializedConfig.PythonPath = pythonPath

	// Choose how many connectors and plugins to make
	const n_connectors = 1
	const n_plugins = 1
	wg := &sync.WaitGroup{}
	wg.Add(n_connectors * n_plugins)

	// Test for issues that might occur during concurrent creation of connectors
	// and deployment of plugins
	// Make a goroutine for each connector
	for j := 0; j < n_connectors; j++ {
		go func() {
			connector, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
			assert.NoError(t, err)

			// Make a goroutine for each plugin
			for k := 0; k < n_plugins; k++ {
				go func() {
					output_id, _, err := RunStep(t, connector, moduleName, stepID, input)
					assert.NoError(t, err)
					assert.Equals(t, output_id, "success")
					//assert.Equals(t,
					//	output_data.(map[interface{}]interface{}),
					//	map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", examplePluginNickname)})
					wg.Done()
				}()
			}
		}()
	}
	// Wait for all the plugins to be done
	wg.Wait()

	t.Cleanup(func() {
		assert.NoError(t, os.Remove(rootDir))
	})
}
