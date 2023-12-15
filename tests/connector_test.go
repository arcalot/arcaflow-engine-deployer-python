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
	"sync"
	"testing"
)

const examplePluginNickname string = "pythonuser"

var inOutConfigGitPullAlways = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"Always"
}
`

var inOutConfigGitPullIfNotPresent = `
{
	"workdir":"/tmp",
	"modulePullPolicy":"IfNotPresent"
}
`

func TestRunStepGit(t *testing.T) {
	moduleName := "arcaflow-plugin-example@git+https://github.com/arcalot/arcaflow-plugin-example.git"
	connector, _ := getConnector(t, inOutConfigGitPullIfNotPresent, nil)
	OutputID, OutputData, Error := RunStep(t, connector, moduleName)
	assert.NoError(t, Error)
	assert.Equals(t, OutputID, "success")
	assert.Equals(t,
		OutputData.(map[interface{}]interface{}),
		map[interface{}]interface{}{"message": fmt.Sprintf("Hello, %s!", examplePluginNickname)})
}

func RunStep(t *testing.T, connector deployer.Connector, moduleName string) (string, any, error) {
	stepID := "hello-world"
	input := map[string]any{
		"name": map[string]any{
			"_type": "nickname",
			"nick":  examplePluginNickname,
		},
	}

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

func TestDeployConcurrent_ConnectorsAndPlugins(t *testing.T) {
	moduleName := "arcaflow-plugin-template-python@git+https://github.com/arcalot/arcaflow-plugin-template-python.git@9b35e855163319963bcc2dbe940a70031a7887c6"
	rootDir := "/tmp"
	var serializedConfig any
	serializedConfig = map[string]any{
		"workdir":          rootDir,
		"modulePullPolicy": "Always",
	}

	factory := pythondeployer.NewFactory()
	deployerSchema := factory.ConfigurationSchema()
	unserializedConfig, err := deployerSchema.UnserializeType(serializedConfig)
	assert.NoError(t, err)

	pythonPath, err := getPythonPath()
	assert.NoError(t, err)
	unserializedConfig.PythonPath = pythonPath

	const n_connectors = 10
	const n_plugins = 3
	wg := &sync.WaitGroup{}
	wg.Add(n_connectors * n_plugins)

	for j := 0; j < n_connectors; j++ {
		go func() {
			c, err := factory.Create(unserializedConfig, log.NewTestLogger(t))
			assert.NoError(t, err)
			for k := 0; k < n_plugins; k++ {
				go func() {
					p, err := c.Deploy(context.Background(), moduleName)
					assert.NoError(t, err)
					assert.NoError(t, p.Close())
					wg.Done()
				}()
			}
		}()
	}

	wg.Wait()
}
